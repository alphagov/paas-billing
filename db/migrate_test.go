package db_test

import (
	"fmt"
	"sort"

	. "github.com/alphagov/paas-billing/db"
	"github.com/alphagov/paas-billing/db/dbhelper"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Migration", func() {

	var (
		sqlClient *PostgresClient
		connstr   string
	)

	BeforeEach(func() {
		var err error
		connstr, err = dbhelper.CreateDB()
		Expect(err).ToNot(HaveOccurred())
		sqlClient, err = NewPostgresClient(connstr)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		err := sqlClient.Close()
		Expect(err).ToNot(HaveOccurred())
		err = dbhelper.DropDB(connstr)
		Expect(err).ToNot(HaveOccurred())
	})

	Specify("schema application is idempotent", func() {
		Expect(sqlClient.InitSchema()).To(Succeed())
		Expect(sqlClient.InitSchema()).To(Succeed())
	})

	Describe("applying migrations", func() {
		var migrationName string

		JustBeforeEach(func() {
			Expect(migrationName).ToNot(BeEmpty())
			priorMigrations, err := migrationSequenceBefore(migrationName)
			Expect(err).ToNot(HaveOccurred())
			Expect(sqlClient.ApplyMigrations(priorMigrations)).To(Succeed())
		})

		Describe("050_add_mb_fields_to_pricing_plans", func() {
			BeforeEach(func() {
				migrationName = "050_add_mb_fields_to_pricing_plans.sql"
			})

			It("should succeed on an empty database", func() {
				err := sqlClient.ApplyMigrations([]string{migrationName})
				Expect(err).NotTo(HaveOccurred())
			})

			Context("when there are existing rows before the migration", func() {
				JustBeforeEach(func() {
					_, err := sqlClient.Conn.Exec(`
						INSERT INTO
							pricing_plans (name, valid_from, plan_guid)
						VALUES
							('medium', '1970-01-01', 'FB0E63F6-E97A-446B-A200-323FC9B562E9')
					`)
					Expect(err).NotTo(HaveOccurred())
				})

				It("should set default values for the new columns", func() {
					err := sqlClient.ApplyMigrations([]string{migrationName})
					Expect(err).NotTo(HaveOccurred())

					var memory_in_mb, storage_in_mb string
					err = sqlClient.Conn.QueryRow(`
						SELECT
							memory_in_mb, storage_in_mb
						FROM pricing_plans
						WHERE plan_guid = 'FB0E63F6-E97A-446B-A200-323FC9B562E9'
					`).Scan(&memory_in_mb, &storage_in_mb)
					Expect(err).NotTo(HaveOccurred())
					Expect(memory_in_mb).To(Equal("0"))
					Expect(storage_in_mb).To(Equal("0"))
				})
			})

			Context("when the migration is applied again on top of existing data", func() {
				JustBeforeEach(func() {
					err := sqlClient.ApplyMigrations([]string{migrationName})
					Expect(err).NotTo(HaveOccurred())

					_, err = sqlClient.Conn.Exec(`
						INSERT INTO
							pricing_plans (name, valid_from, plan_guid, memory_in_mb, storage_in_mb)
						VALUES
							('medium', '1970-01-01', 'FB0E63F6-E97A-446B-A200-323FC9B562E9', 10240, 102400)
					`)
					Expect(err).NotTo(HaveOccurred())
				})

				It("should preserve existing values", func() {
					err := sqlClient.ApplyMigrations([]string{migrationName})
					Expect(err).NotTo(HaveOccurred())

					var memory_in_mb, storage_in_mb string
					err = sqlClient.Conn.QueryRow(`
						SELECT
							memory_in_mb, storage_in_mb
						FROM pricing_plans
						WHERE plan_guid = 'FB0E63F6-E97A-446B-A200-323FC9B562E9'
					`).Scan(&memory_in_mb, &storage_in_mb)
					Expect(err).NotTo(HaveOccurred())
					Expect(memory_in_mb).To(Equal("10240"))
					Expect(storage_in_mb).To(Equal("102400"))
				})
			})
		})
	})
})

func migrationSequenceBefore(upToButNotIncluding string) ([]string, error) {
	sortedMigrations, err := MigrationSequence()
	if err != nil {
		return nil, err
	}

	index := sort.SearchStrings(sortedMigrations, upToButNotIncluding)
	if index == len(sortedMigrations) {
		return nil, fmt.Errorf("migration {} was not found", upToButNotIncluding)
	}
	return sortedMigrations[:index], nil
}
