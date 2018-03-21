package db_test

import (
	"database/sql"
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

		Describe("051_create_resource_usage", func() {
			var (
				appGuid, serviceInstanceGuid, planGuid string
			)

			BeforeEach(func() {
				migrationName = "051_create_resource_usage.sql"
				appGuid = "000007d7-8a78-4cc0-9be3-b41f89460ae8"
				serviceInstanceGuid = "eb3eb3ae-0fb6-475e-af93-975e80f6361a"
				planGuid = "FB0E63F6-E97A-446B-A200-323FC9B562E9"
			})

			It("should succeed on an empty database", func() {
				err := sqlClient.ApplyMigrations([]string{migrationName})
				Expect(err).NotTo(HaveOccurred())
			})

			Context("when combining app and service events", func() {
				JustBeforeEach(func() {
					_, err := sqlClient.Conn.Exec(`
						INSERT INTO
							pricing_plans (name, valid_from, plan_guid, memory_in_mb, storage_in_mb, number_of_nodes)
						VALUES
							('medium', '1970-01-01', $1, 10240, 102400, 1)
					`, planGuid)
					Expect(err).NotTo(HaveOccurred())

					// Insert an app STARTED and STOPPED event
					_, err = sqlClient.Conn.Exec(`
						INSERT INTO
							app_usage_events (created_at, guid, raw_message)
						VALUES
							('2018-01-01 00:00:00', 'a', '{"state": "STARTED", "app_guid": "` + appGuid + `", "app_name": "app1", "org_guid": "org_guid", "task_guid": null, "task_name": null, "space_guid": "space_guid", "space_name": "space1", "process_type": "web", "package_state": "PENDING", "buildpack_guid": null, "buildpack_name": null, "instance_count": 1, "previous_state": "STOPPED", "parent_app_guid": "parent_app_guid", "parent_app_name": "app_parent", "previous_package_state": "UNKNOWN", "previous_instance_count": 1, "memory_in_mb_per_instance": 1024, "previous_memory_in_mb_per_instance": 1024}'::jsonb),
							('2018-01-01 01:00:00', 'b', '{"state": "STOPPED", "app_guid": "` + appGuid + `", "app_name": "app1", "org_guid": "org_guid", "task_guid": null, "task_name": null, "space_guid": "space_guid", "space_name": "space1", "process_type": "web", "package_state": "PENDING", "buildpack_guid": null, "buildpack_name": null, "instance_count": 1, "previous_state": "STARTED", "parent_app_guid": "parent_app_guid", "parent_app_name": "app_parent", "previous_package_state": "UNKNOWN", "previous_instance_count": 1, "memory_in_mb_per_instance": 1024, "previous_memory_in_mb_per_instance": 1024}'::jsonb)
					`)
					Expect(err).NotTo(HaveOccurred())

					// Insert a service CREATED and DELETED event
					_, err = sqlClient.Conn.Exec(`
						INSERT INTO
							service_usage_events (created_at, guid, raw_message)
						VALUES
							('2018-01-01 00:00:00', '1', '{"state": "CREATED", "org_guid": "org_guid", "space_guid": "space_guid", "space_name": "sandbox", "service_guid": "efadb775-58c4-4e17-8087-6d0f4febc489", "service_label": "postgres", "service_plan_guid": "` + planGuid + `", "service_plan_name": "Free", "service_instance_guid": "` + serviceInstanceGuid + `", "service_instance_name": "ja-rails-postgres", "service_instance_type": "managed_service_instance"}'::jsonb),
							('2018-01-01 01:00:00', '2', '{"state": "DELETED", "org_guid": "org_guid", "space_guid": "space_guid", "space_name": "sandbox", "service_guid": "efadb775-58c4-4e17-8087-6d0f4febc489", "service_label": "postgres", "service_plan_guid": "` + planGuid + `", "service_plan_name": "Free", "service_instance_guid": "` + serviceInstanceGuid + `", "service_instance_name": "ja-rails-postgres", "service_instance_type": "managed_service_instance"}'::jsonb)
					`)
					Expect(err).NotTo(HaveOccurred())
				})

				It("generates resource durations for both", func() {
					err := sqlClient.ApplyMigrations([]string{migrationName})
					Expect(err).NotTo(HaveOccurred())

					var count int
					err = sqlClient.Conn.QueryRow(`
						SELECT COUNT(*) FROM resource_usage
					`).Scan(&count)
					Expect(err).NotTo(HaveOccurred())
					Expect(count).To(Equal(2))
				})

				It("extracts memory, storage & instance count from events and sets default values", func() {
					err := sqlClient.ApplyMigrations([]string{migrationName})
					Expect(err).NotTo(HaveOccurred())

					var memory_in_mb, storage_in_mb, number_of_nodes string
					err = sqlClient.Conn.QueryRow(`
						SELECT
							memory_in_mb, storage_in_mb, number_of_nodes
						FROM
							resource_usage
						WHERE
							resource_guid = $1
					`, appGuid).Scan(&memory_in_mb, &storage_in_mb, &number_of_nodes)
					Expect(err).NotTo(HaveOccurred())
					Expect(memory_in_mb).To(Equal("1024"))
					Expect(storage_in_mb).To(Equal("0"))
					Expect(number_of_nodes).To(Equal("1"))
				})

				It("sets memory and storage to NULL for service resource durations", func() {
					err := sqlClient.ApplyMigrations([]string{migrationName})
					Expect(err).NotTo(HaveOccurred())

					var memory_in_mb, storage_in_mb sql.NullInt64
					err = sqlClient.Conn.QueryRow(`
						SELECT
							memory_in_mb, storage_in_mb
						FROM
							resource_usage
						WHERE
							resource_guid = $1
					`, serviceInstanceGuid).Scan(&memory_in_mb, &storage_in_mb)
					Expect(err).NotTo(HaveOccurred())
					Expect(memory_in_mb.Valid).To(BeFalse())
					Expect(storage_in_mb.Valid).To(BeFalse())
				})
			})

			Context("usage rows with no durations are discarded", func() {
				JustBeforeEach(func() {

					// Insert an a single app STARTED and STOPPED event
					// Insert a series of STARTED events and a STOPPED event all with the same created_at
					_, err := sqlClient.Conn.Exec(`
						INSERT INTO
							app_usage_events (created_at, guid, raw_message)
						VALUES
							('2018-01-01 00:00:00', 'a', '{"state": "STARTED", "app_guid": "` + appGuid + `", "app_name": "app1", "org_guid": "org_guid", "task_guid": null, "task_name": null, "space_guid": "space_guid", "space_name": "space1", "process_type": "web", "package_state": "PENDING", "buildpack_guid": null, "buildpack_name": null, "instance_count": 1, "previous_state": "STOPPED", "parent_app_guid": "parent_app_guid", "parent_app_name": "app_parent", "previous_package_state": "UNKNOWN", "previous_instance_count": 1, "memory_in_mb_per_instance": 1024, "previous_memory_in_mb_per_instance": 1024}'::jsonb),
							('2018-01-01 01:00:00', 'b', '{"state": "STOPPED", "app_guid": "` + appGuid + `", "app_name": "app1", "org_guid": "org_guid", "task_guid": null, "task_name": null, "space_guid": "space_guid", "space_name": "space1", "process_type": "web", "package_state": "PENDING", "buildpack_guid": null, "buildpack_name": null, "instance_count": 1, "previous_state": "STARTED", "parent_app_guid": "parent_app_guid", "parent_app_name": "app_parent", "previous_package_state": "UNKNOWN", "previous_instance_count": 1, "memory_in_mb_per_instance": 1024, "previous_memory_in_mb_per_instance": 1024}'::jsonb),
							('2018-01-01 10:00:00', 'c', '{"state": "STARTED", "app_guid": "` + appGuid + `", "app_name": "app1", "org_guid": "org_guid", "task_guid": null, "task_name": null, "space_guid": "space_guid", "space_name": "space1", "process_type": "web", "package_state": "PENDING", "buildpack_guid": null, "buildpack_name": null, "instance_count": 1, "previous_state": "STOPPED", "parent_app_guid": "parent_app_guid", "parent_app_name": "app_parent", "previous_package_state": "UNKNOWN", "previous_instance_count": 1, "memory_in_mb_per_instance": 1024, "previous_memory_in_mb_per_instance": 1024}'::jsonb),
							('2018-01-01 10:00:00', 'd', '{"state": "STARTED", "app_guid": "` + appGuid + `", "app_name": "app1", "org_guid": "org_guid", "task_guid": null, "task_name": null, "space_guid": "space_guid", "space_name": "space1", "process_type": "web", "package_state": "PENDING", "buildpack_guid": null, "buildpack_name": null, "instance_count": 1, "previous_state": "STOPPED", "parent_app_guid": "parent_app_guid", "parent_app_name": "app_parent", "previous_package_state": "UNKNOWN", "previous_instance_count": 1, "memory_in_mb_per_instance": 1024, "previous_memory_in_mb_per_instance": 1024}'::jsonb),
							('2018-01-01 10:00:00', 'e', '{"state": "STARTED", "app_guid": "` + appGuid + `", "app_name": "app1", "org_guid": "org_guid", "task_guid": null, "task_name": null, "space_guid": "space_guid", "space_name": "space1", "process_type": "web", "package_state": "PENDING", "buildpack_guid": null, "buildpack_name": null, "instance_count": 1, "previous_state": "STOPPED", "parent_app_guid": "parent_app_guid", "parent_app_name": "app_parent", "previous_package_state": "UNKNOWN", "previous_instance_count": 1, "memory_in_mb_per_instance": 1024, "previous_memory_in_mb_per_instance": 1024}'::jsonb),
							('2018-01-01 10:00:00', 'f', '{"state": "STOPPED", "app_guid": "` + appGuid + `", "app_name": "app1", "org_guid": "org_guid", "task_guid": null, "task_name": null, "space_guid": "space_guid", "space_name": "space1", "process_type": "web", "package_state": "PENDING", "buildpack_guid": null, "buildpack_name": null, "instance_count": 1, "previous_state": "STARTED", "parent_app_guid": "parent_app_guid", "parent_app_name": "app_parent", "previous_package_state": "UNKNOWN", "previous_instance_count": 1, "memory_in_mb_per_instance": 1024, "previous_memory_in_mb_per_instance": 1024}'::jsonb)
					`)
					Expect(err).NotTo(HaveOccurred())
				})

				It("only outputs a single resource row because the others have zero duration", func() {
					err := sqlClient.ApplyMigrations([]string{migrationName})
					Expect(err).NotTo(HaveOccurred())

					var count int
					err = sqlClient.Conn.QueryRow(`
						SELECT COUNT(*) FROM resource_usage
					`).Scan(&count)
					Expect(err).NotTo(HaveOccurred())
					Expect(count).To(Equal(1))
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
