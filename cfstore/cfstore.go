package cfstore

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"code.cloudfoundry.org/lager"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/lib/pq"
)

const (
	DefaultInitTimeout = 5 * time.Minute
)

type Config struct {
	// CFClient config
	ClientConfig *cfclient.Config
	// Client for communicating with cf
	Client CFDataClient
	// Database connection
	DB *sql.DB
	// Logger overrides the default logger
	Logger lager.Logger
	// Collection delay
	Schedule time.Duration
}

type Store struct {
	client CFDataClient
	db     *sql.DB
	logger lager.Logger
}

func (s *Store) Init() error {
	s.logger.Info("initializing")
	ctx, cancel := context.WithTimeout(context.Background(), DefaultInitTimeout)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := s.execFile(tx, "create_services.sql"); err != nil {
		return err
	}
	if err := s.execFile(tx, "create_service_plans.sql"); err != nil {
		return err
	}
	if err := s.collectServices(tx); err != nil {
		return err
	}
	if err := s.collectServicePlans(tx); err != nil {
		return err
	}
	s.logger.Info("initialized")
	return tx.Commit()
}

func (s *Store) execFile(tx *sql.Tx, filename string) error {
	schemaFilename := schemaFile(filename)
	sql, err := ioutil.ReadFile(schemaFilename)
	if err != nil {
		return fmt.Errorf("failed to execute sql file %s: %s", schemaFilename, err)
	}
	s.logger.Info("executing-sql", lager.Data{
		"filename": filename,
	})
	_, err = tx.Exec(string(sql))
	if err != nil {
		return wrapPqError(err, schemaFilename)
	}
	return nil
}

func (s *Store) CollectServicePlans() error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultInitTimeout)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := s.collectServicePlans(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) collectServicePlans(tx *sql.Tx) error {
	plans, err := s.client.ListServicePlans()
	if err != nil {
		return err
	}
	for _, plan := range plans {
		validFrom := plan.UpdatedAt
		var planCount int
		err := tx.QueryRow(`
			select count(*)
			from service_plans
			where guid = $1
		`, plan.Guid).Scan(&planCount)
		if err != nil {
			return err
		}
		if planCount == 0 {
			validFrom = plan.CreatedAt
		}

		var serviceValidFrom *time.Time
		err = tx.QueryRow(`
			select valid_from
			from services
			where guid = $1
			order by valid_from desc
			limit 1
		`, plan.ServiceGuid).Scan(&serviceValidFrom)
		if err == sql.ErrNoRows {
			panic("meh dunno what to do")
		} else if err != nil {
			return err
		}

		_, err = tx.Exec(`
			insert into service_plans (
				guid, valid_from,
				name, description,
				unique_id,
				active, public, free,
				extra,
				created_at, updated_at,
				service_guid, service_valid_from
			) values (
				$1, $2,
				$3, $4,
				$5,
				$6, $7, $8,
				$9,
				$10, $11,
				$12, $13
			) on conflict (guid, valid_from) do nothing
		`, plan.Guid, validFrom,
			plan.Name, plan.Description,
			plan.UniqueId,
			plan.Active, plan.Public, plan.Free,
			plan.Extra,
			plan.CreatedAt, plan.UpdatedAt,
			plan.ServiceGuid, serviceValidFrom)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) CollectServices() error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultInitTimeout)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := s.collectServices(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) collectServices(tx *sql.Tx) error {
	services, err := s.client.ListServices()
	if err != nil {
		return err
	}
	for _, service := range services {
		validFrom := service.UpdatedAt
		var serviceCount int
		err := tx.QueryRow(`
			select count(*)
			from services
			where guid = $1
		`, service.Guid).Scan(&serviceCount)
		if err != nil {
			return err
		}
		if serviceCount == 0 {
			validFrom = service.CreatedAt
		}

		_, err = tx.Exec(`
			insert into services (
				guid, valid_from,
				label, description,
				active, bindable,
				service_broker_guid,
				created_at, updated_at
			) values (
				$1, $2,
				$3, $4,
				$5, $6,
				$7,
				$8, $9
			) on conflict (guid, valid_from) do nothing
		`, service.Guid, validFrom,
			service.Label, service.Description,
			service.Active, service.Bindable,
			service.ServiceBrokerGuid,
			service.CreatedAt, service.UpdatedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

func wrapPqError(err error, prefix string) error {
	msg := err.Error()
	if err, ok := err.(*pq.Error); ok {
		msg = err.Message
		if err.Detail != "" {
			msg += ": " + err.Detail
		}
		if err.Hint != "" {
			msg += ": " + err.Hint
		}
		if err.Where != "" {
			msg += ": " + err.Where
		}
	}
	return fmt.Errorf("%s: %s", prefix, msg)
}

func schemaDir() string {
	root := os.Getenv("APP_ROOT")
	if root == "" {
		root = os.Getenv("PWD")
	}
	if root == "" {
		root, _ = os.Getwd()
	}
	return filepath.Join(root, "cfstore", "sql")
}

func schemaFile(filename string) string {
	return filepath.Join(schemaDir(), filename)
}

func New(cfg Config) (*Store, error) {
	if cfg.Logger == nil {
		cfg.Logger = lager.NewLogger("historic-data-store")
	}
	store := &Store{
		client: cfg.Client,
		logger: cfg.Logger,
		db:     cfg.DB,
	}
	return store, nil
}
