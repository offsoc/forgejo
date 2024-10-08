# Integration tests

Thank you for your effort to provide good software tests for Forgejo.
Please also read the general testing instructions in the
[Forgejo contributor documentation](https://forgejo.org/docs/next/contributor/testing/).

This file is meant to provide specific information for the integration tests
as well as some tips and tricks you should know.

Feel free to extend this file with more instructions if you feel like you have something to share!


## How to run the tests?

Before running any tests, please ensure you perform a clean build:

```
make clean build
```

Integration tests can be run with make commands for the
appropriate backends, namely:
```shell
make test-sqlite
make test-pgsql
make test-mysql
```


### Run tests via local forgejo runner

If you have a [forgejo runner](https://code.forgejo.org/forgejo/runner/),
you can use it to run the test jobs:

#### Run all jobs

```
forgejo-runner exec -W .forgejo/workflows/testing.yml --event=pull_request
```

Warning: This file defines many jobs, so it will be resource-intensive and therefore not recommended.

#### Run single job

```SHELL
forgejo-runner exec -W .forgejo/workflows/testing.yml --event=pull_request -j <job_name>
```

You can list all job names via:

```SHELL
forgejo-runner exec -W .forgejo/workflows/testing.yml --event=pull_request -l
```

### Run sqlite integration tests
Start tests
```
make test-sqlite
```

### Run MySQL integration tests
Setup a MySQL database inside docker
```
docker run -e "MYSQL_DATABASE=test" -e "MYSQL_ALLOW_EMPTY_PASSWORD=yes" -p 3306:3306 --rm --name mysql mysql:latest #(just ctrl-c to stop db and clean the container)
docker run -p 9200:9200 -p 9300:9300 -e "discovery.type=single-node" --rm --name elasticsearch elasticsearch:7.6.0 #(in a second terminal, just ctrl-c to stop db and clean the container)
```
Start tests based on the database container
```
TEST_MYSQL_HOST=localhost:3306 TEST_MYSQL_DBNAME=test TEST_MYSQL_USERNAME=root TEST_MYSQL_PASSWORD='' make test-mysql
```

### Run pgsql integration tests
Setup a pgsql database inside docker
```
docker run -e "POSTGRES_DB=test" -p 5432:5432 --rm --name pgsql postgres:latest #(just ctrl-c to stop db and clean the container)
```
Start tests based on the database container
```
TEST_PGSQL_HOST=localhost:5432 TEST_PGSQL_DBNAME=test TEST_PGSQL_USERNAME=postgres TEST_PGSQL_PASSWORD=postgres make test-pgsql
```

### Running individual tests

Example command to run GPG test:

For SQLite:

```
make test-sqlite#GPG
```

For other databases (replace `mysql` to `pgsql`):

```
TEST_MYSQL_HOST=localhost:1433 TEST_MYSQL_DBNAME=test TEST_MYSQL_USERNAME=sa TEST_MYSQL_PASSWORD=MwantsaSecurePassword1 make test-mysql#GPG
```

## Setting timeouts for declaring long-tests and long-flushes

We appreciate that some testing machines may not be very powerful and
the default timeouts for declaring a slow test or a slow clean-up flush
may not be appropriate.

You can either:

* Within the test ini file set the following section:

```ini
[integration-tests]
SLOW_TEST = 10s ; 10s is the default value
SLOW_FLUSH = 5S ; 5s is the default value
```

* Set the following environment variables:

```bash
GITEA_SLOW_TEST_TIME="10s" GITEA_SLOW_FLUSH_TIME="5s" make test-sqlite
```

## Tips and tricks

If you know noteworthy tests that can act as an inspiration for new tests,
please add some details here.
