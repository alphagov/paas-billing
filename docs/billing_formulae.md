# Billing formulae


## About

At its core, Paas billing uses a series of formulae to calculate the bills for services and resources provisioned on behalf of Paas tenants. This document summarises these formulae.

These formulae are also present in JSON configuration files in the [paas-cf](https://github.com/alphagov/paas-cf) repository but this is a useful summary to avoid having to extract all the information from this repository.

Please note, in this document we are only looking at _how_ the bills are calculated (the formula used) not the actual hourly prices applied. The latter can be obtained from the [AWS pricing website](https://aws.amazon.com/pricing/).

The formulae below are written in SQL for the Postgres database. In the formulae, the provider_price is the price charged by AWS or Aiven, typically per hour, for the service.

[`CEIL`](https://www.postgresql.org/docs/current/functions-math.html) is a Postgres function that rounds up to the nearest whole number (e.g. `CEIL(1.0000001) = 2`).


## List of resources billed by Paas

### eu-west-2

- RDS Postgres
- RDS MySql
- S3
- Cloudfront cdn route
- Elasticsearch
- MongoDB
- InfluxDB
- Redis
- Prometheus
- Staging
- Task
- Service

### eu-west-1

- RDS Postgres
- RDS MySql
- S3
- Cloudfront cdn route
- Elasticsearch
- InfluxDB
- MongoDB
- Redis
- Prometheus
- Service
- Staging
- Task


## How each resource is billed


### AWS region eu-west-2


#### RDS Postgres

##### Storage

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - infinity | (storage_in_mb/1024) * CEIL(time_in_seconds/2678401) * provider_price |

##### Instance

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - infinity | CEIL(time_in_seconds/3600) * provider_price |


#### RDS MySql

##### Storage

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - infinity | (storage_in_mb/1024) * CEIL(time_in_seconds/2678401) * provider_price |

##### Instance

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - infinity | CEIL(time_in_seconds/3600) * provider_price |


#### S3

| Date range | Formula |
| :--: | :--: |
| 2019-02-01 - infinity | CEIL(time_in_seconds/3600) * provider_price |


#### Cloudfront cdn route

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - infinity | Zero priced |


#### Elasticsearch

##### elasticsearch tiny (compose)

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - infinity | ((1936.57/(48*1024))/30/24) * memory_in_mb * CEIL(time_in_seconds / 3600) |

##### All other Elasticsearch

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - infinity | CEIL(time_in_seconds/3600) * provider_price |


#### MongoDB tiny (compose)

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - infinity | ((1936.57/(48*1024))/30/24) * memory_in_mb * CEIL(time_in_seconds / 3600) |


#### InfluxDB

| Date range | Formula |
| :--: | :--: |
| 2019-11-01 - infinity | CEIL(time_in_seconds/3600) * provider_price |

Reference: https://aiven.io/influxdb

#### Redis

##### redis tiny (compose)

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - infinity | ((1936.57/(48*1024))/30/24) * memory_in_mb * CEIL(time_in_seconds / 3600) |

##### Other redis

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - infinity | number_of_nodes * CEIL(time_in_seconds/3600) * provider_price |


#### Prometheus

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - infinity | Zero priced |


#### Staging

##### Before 2019-03-01 (2017-01-01 to 2019-03-01)

###### Platform

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - 2019-03-01 00:00:00 | (number_of_nodes * CEIL(time_in_seconds / 3600) * (memory_in_mb/1024.0) * 0.01) * provider_price |
| 2019-03-01 00:00:00 - infinity | (number_of_nodes * time_in_seconds * (memory_in_mb/1024.0) * (0.01 / 3600)) * provider_price |

###### Instance

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - 2019-03-01 00:00:00 | number_of_nodes * CEIL(time_in_seconds / 3600) * (memory_in_mb/1024.0) * 0.01 |
| 2019-03-01 00:00:00 - infinity | number_of_nodes * time_in_seconds * (memory_in_mb/1024.0) * (0.01 / 3600) |


#### Task

##### Before 2019-03-01

###### Platform

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - 2019-03-01 00:00:00 | (number_of_nodes * CEIL(time_in_seconds / 3600) * (memory_in_mb/1024.0) * 0.01) * provider_price |
| 2019-03-01 00:00:00 - infinity | (number_of_nodes * time_in_seconds * (memory_in_mb/1024.0) * (0.01 / 3600)) * provider_price |

###### Instance

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - 2019-03-01 00:00:00 | number_of_nodes * CEIL(time_in_seconds / 3600) * (memory_in_mb/1024.0) * 0.01 |
| 2019-03-01 00:00:00 - infinity | number_of_nodes * time_in_seconds * (memory_in_mb/1024.0) * (0.01 / 3600) |


#### Service

| Date range | Formula |
| :--: | :--: |
| 1970-01-01 - infinity | Zero priced |



### AWS region eu-west-1



#### RDS Postgres

##### Storage

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - infinity | (storage_in_mb/1024) * CEIL(time_in_seconds/2678401) * provider_price |

##### Instance

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - infinity | CEIL(time_in_seconds/3600) * provider_price |


#### RDS MySql

##### Storage

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - infinity | (storage_in_mb/1024) * CEIL(time_in_seconds/2678401) * provider_price |

##### Instance

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - infinity | CEIL(time_in_seconds/3600) * provider_price |


#### S3

| Date range | Formula |
| :--: | :--: |
| 2019-02-01 - infinity | CEIL(time_in_seconds/3600) * provider_price |


#### Cloudfront cdn route

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - infinity | Zero priced |


#### Elasticsearch

##### elasticsearch tiny (compose)

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - infinity | ((1936.57/(48*1024))/30/24) * memory_in_mb * CEIL(time_in_seconds / 3600) |

##### All other Elasticsearch

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - infinity | CEIL(time_in_seconds/3600) * provider_price |


#### InfluxDB

| Date range | Formula |
| :--: | :--: |
| 2019-11-01 - infinity | CEIL(time_in_seconds/3600) * provider_price |

Reference: https://aiven.io/influxdb

#### MongoDB tiny (compose)

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - infinity | ((1936.57/(48*1024))/30/24) * memory_in_mb * CEIL(time_in_seconds / 3600) |


#### Redis

##### redis tiny (compose)

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - infinity | ((1936.57/(48*1024))/30/24) * memory_in_mb * CEIL(time_in_seconds / 3600) |

##### Other redis

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - infinity | number_of_nodes * CEIL(time_in_seconds/3600) * provider_price |


#### Prometheus

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - infinity | Zero priced |


#### Service

| Date range | Formula |
| :--: | :--: |
| 1970-01-01 - infinity | Zero priced |


#### Staging

##### Before 2019-03-01

###### Platform

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - 2019-03-01 00:00:00 | (number_of_nodes * CEIL(time_in_seconds / 3600) * (memory_in_mb/1024.0) * 0.01) * provider_price |
| 2019-03-01 00:00:00 - infinity | (number_of_nodes * time_in_seconds * (memory_in_mb/1024.0) * (0.01 / 3600)) * provider_price |

###### Instance

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - 2019-03-01 00:00:00 | number_of_nodes * CEIL(time_in_seconds / 3600) * (memory_in_mb/1024.0) * 0.01 |
| 2019-03-01 00:00:00 - infinity | number_of_nodes * time_in_seconds * (memory_in_mb/1024.0) * (0.01 / 3600) |


#### Task

##### Before 2019-03-01

###### Platform

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - 2019-03-01 00:00:00 | (number_of_nodes * CEIL(time_in_seconds / 3600) * (memory_in_mb/1024.0) * 0.01) * provider_price |
| 2019-03-01 00:00:00 - infinity | (number_of_nodes * time_in_seconds * (memory_in_mb/1024.0) * (0.01 / 3600)) * provider_price |

###### Instance

| Date range | Formula |
| :--: | :--: |
| 2017-01-01 - 2019-03-01 00:00:00 | number_of_nodes * CEIL(time_in_seconds / 3600) * (memory_in_mb/1024.0) * 0.01 |
| 2019-03-01 00:00:00 - infinity | number_of_nodes * time_in_seconds * (memory_in_mb/1024.0) * (0.01 / 3600) |
