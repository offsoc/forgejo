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
docker run -e MYSQL_DATABASE=test -e MYSQL_ALLOW_EMPTY_PASSWORD=yes -p 3306:3306 --rm --name mysql mysql:latest #(Ctrl-c to stop the database)
```
Start tests based on the database container
```
TEST_MYSQL_HOST=localhost:3306 TEST_MYSQL_DBNAME=test?multiStatements=true TEST_MYSQL_USERNAME=root TEST_MYSQL_PASSWORD='' make test-mysql
```

### Run pgsql integration tests
Setup a pgsql database inside docker
```
docker run -e "POSTGRES_DB=test" -e POSTGRES_PASSWORD=postgres -p 5432:5432 --rm --name pgsql postgres:latest #(Ctrl-c to stop the database)
```
Start tests based on the database container
```
TEST_STORAGE_TYPE=local TEST_PGSQL_HOST=localhost:5432 TEST_PGSQL_DBNAME=test TEST_PGSQL_USERNAME=postgres TEST_PGSQL_PASSWORD=postgres make test-pgsql
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


# Example: Test app.ini parameter with effect on visible elements

## The test file and where to find it

The file is defined in:
```
forgejo/tests/integration/disable_forgotten_password_test.go
```
and the test function is called:
```
TestDisableForgottenPasswordTrue
```

## How to run the isolated test

Now we know the name of the test function (well, we defined it). We can run a selected test function via:


```
make clean build
make test-sqlite#TestDisableForgottenPasswordTrue
```

## The parameter we want to manipulate

The first line of interest is
```
defer test.MockVariableValue(&setting.Service.SignInForgottenPasswordEnabled, true)()
```

Here we set the app.ini parameter 
```
[Service]
SIGNIN_FORGOTTEN_PASSWORD_ENABLED
```
to true.

As you can see, the test doesn't include the processing of the actual processing of the value entered into the app.ini. We start your test after modules/setting/service.go did this:

```
Service.SignInForgottenPasswordEnabled = sec.Key("SIGNIN_FORGOTTEN_PASSWORD_ENABLED").MustBool(true)
```

In the case SignInForgottenPasswordEnabled doesn't exist in modules/setting/service.go (e.g. by misspelling it or if it just doesn't exists), you will get an error when running the test. i.e. the test will not run. 


## Get the page 

Our parameter changes elements on the /user/login page. Thus we want to see what happens there and we need to grep it:

```
req := NewRequest(t, "GET", "/user/login/")
resp := MakeRequest(t, req, http.StatusOK)
```

Note: You can add "fmt" to the import section and then use e.g.

```
fmt.Printf("XXXXX %+v\n",resp)
```
to "visually" inspect what is going on (here the variable resp). I add the tag XXXXX because this help to visually find the output as well as allow to search for it easily. Or %T to see the type of a variable:
```
parser := NewHTMLParser(t, resp.Body)
fmt.Printf("XXXXX %T\n",parser)
```
This tells us that the parser is related to integration.HTMLDoc. (we can google this "go integration.HTMLDoc" for more information)

## Parse the page and find selectors

We find more information about integration.HTMLDoc here: https://pkg.go.dev/code.gitea.io/gitea/tests/integration

More specifically we are interested in .Find() https://pkg.go.dev/code.gitea.io/gitea/tests/integration#HTMLDoc.Find

After we looked into resp, we know that the target of our interest lies in the lines:

```
<div class="field">
    <a href="/user/forgot_password">Forgot password?</a>
</div>
```

Hence we ask the parser to find all instances of a link:
```
doc := NewHTMLParser(t, resp.Body).Find("a")
```

Obviously, there will be more than one link on the page. Thus we will inspect them individually via a for loop:
```
for i := range doc.Nodes {
   one_element := doc.Eq(i)
}
```

one_element is of "type Selection" ( more information how the methods type has see here: https://pkg.go.dev/github.com/PuerkitoBio/goquery#Selection )

For example we can list the value which is in between "<a>" and "</a>" with

```
doc := NewHTMLParser(t, resp.Body).Find("a")
for i := range doc.Nodes {
    oneElement := doc.Eq(i)
    htmlText, _ := oneElement.Html()
    fmt.Printf("XXXXX %+v\n",htmlText)
}
```
For our problem we could now check for "Forgot password?" which is in between "<a>" "</a>". But this is theoretically depended on the language setting and also not the functionality we want to check against. Hence, we will focus on the link itself. We get the link information via the .Attr() method: 

```
doc := NewHTMLParser(t, resp.Body).Find("a")
for i := range doc.Nodes {
    oneElement := doc.Eq(i)
    attValue, attExists := oneElement.Attr("href")
    if attExists {
        fmt.Printf("XXXXX %+v\n",attValue)
    }
}
```

This results in:

```
XXXXX /
XXXXX /explore/repos
XXXXX https://forgejo.org/docs/latest/
XXXXX /user/sign_up
XXXXX /user/login
XXXXX /user/sign_up
XXXXX /user/forgot_password
XXXXX https://forgejo.org
XXXXX /assets/licenses.txt
XXXXX /api/swagger
```

Now we count the number of instances of "/user/forgot_password" and evaluate that.

```
package integration

import (
        "net/http"
        "testing"

        "code.gitea.io/gitea/modules/setting"
        "code.gitea.io/gitea/modules/test"
        "code.gitea.io/gitea/tests"

        "github.com/stretchr/testify/assert"
)

func TestDisableForgottenPasswordTrue(t *testing.T) {
        defer tests.PrepareTestEnv(t)()
        defer test.MockVariableValue(&setting.Service.SignInForgottenPasswordEnabled, true)()

        req := NewRequest(t, "GET", "/user/login/")
        resp := MakeRequest(t, req, http.StatusOK)
        doc := NewHTMLParser(t, resp.Body).Find("a")
        var counterInstances int = 0
        for i := range doc.Nodes {
            oneElement := doc.Eq(i)
            attValue, attExists := oneElement.Attr("href")
            if attExists {
                if attValue == "/user/forgot_password" {
                    counterInstances += 1
                }
            }
        }
        assert.EqualValues(t, 1, counterInstances)
}
```


One is the loneliest number because it is the only correct solution here.

```
assert.EqualValues(t, 1, counterInstances)
```

Note: Other options for assert can be found here: https://pkg.go.dev/github.com/stretchr/testify/assert

## Adding the two other cases

We make two copies of

```
func TestDisableForgottenPasswordTrue(t *testing.T)
```

and call them

```
func TestDisableForgottenPasswordDefault(t *testing.T)
```
and
```
func TestDisableForgottenPasswordFalse(t *testing.T)
```


In
```
func TestDisableForgottenPasswordDefault(t *testing.T)
```
we remove the line
```
defer test.MockVariableValue(&setting.Service.SignInForgottenPasswordEnabled, true)()
```
to test the default setting.


In
```
func TestDisableForgottenPasswordFalse(t *testing.T)
```
we change the two lines:
```
defer test.MockVariableValue(&setting.Service.SignInForgottenPasswordEnabled, false)()
```
(parameter is false)
and
```
assert.EqualValues(t, 0, counterInstances)
```
(the link never occurs i.e. counterInstances is zero).


## Final test
Check if everything works during the real test:
```
make test-sqlite
```

