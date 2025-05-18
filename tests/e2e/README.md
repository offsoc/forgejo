# End to end tests

Thank you for your effort to provide good software tests for Forgejo.
Please also read the general testing instructions in the
[Forgejo contributor documentation](https://forgejo.org/docs/next/contributor/testing/)
and make sure to also check the
[Playwright documentation](https://playwright.dev/docs/intro)
for further information.

This file is meant to provide specific information for the integration tests
as well as some tips and tricks you should know.

Feel free to extend this file with more instructions if you feel like you have something to share!


## How to run the tests?

Before running any tests, please ensure you perform a clean frontend build:

```
make clean frontend
```

Whenever you modify frontend code (i.e. JavaScript and CSS files),
you need to create a new frontend build.

For tests that require interactive Git repos,
you also need to ensure a Forgejo binary is ready to be used by Git hooks.
For this, you additionally need to run

~~~
make TAGS="sqlite sqlite_unlock_notify" backend
~~~

### Install dependencies

Browsertesting is performed by playwright.
You need certain system libraries and playwright will download required browsers.
Playwright takes care of this when you run:

```
npx playwright install-deps
```

> **Note**
> On some operating systems, the installation of missing libraries can complicate testing certain browsers.
> It is often not necessary to test with all browsers locally.
> Choosing either Firefox or Chromium is fine.


### Run all tests

If you want to run the full test suite, you can use

```
make test-e2e-sqlite
```

### Interactive testing

We recommend that you use interactive testing for the development.
After you performed the required builds,
you should use one shell to start the debugserver (and leave it running):

```
make test-e2e-debugserver
```

It allows you to explore the test data in your local browser,
and playwright to perform tests on it.

> **Note**
> The modifications persist while the debugserver is running.
> If you modified things, it might be useful to restart it to get back to a fresh state.
> While writing playwright tests, you either
> need to ensure they are resilient against repeated runs
> (e.g. when only creating new content),
> or that they restore the initial state for the next browser run.

#### With the playwright UI:

Playwright ships with an integrated UI mode which allows you to
run individual tests and to debug them by seeing detailed traces of what playwright does.
Launch it with:

```
npx playwright test --ui
```

#### Running individual tests

```
npx playwright test actions.test.e2e.ts:9
```

First, specify the complete test filename,
and after the colon you can put the linenumber where the test is defined.


#### With VSCodium or VSCode

To debug a test, you can also use "Playwright Test" for
[VScodium](https://open-vsx.org/extension/ms-playwright/playwright)
or [VSCode](https://marketplace.visualstudio.com/items?itemName=ms-playwright.playwright).


### Run all tests via local act_runner

If you have a [forgejo runner](https://code.forgejo.org/forgejo/runner/),
you can use it to run the test jobs:

```
forgejo-runner exec -W .forgejo/workflows/testing.yml -j test-e2e
```

Note that the CI workflow has some logic to run tests based on changed files only.
This might conflict with your local setup and not run all the desired tests
because it might only look at file changes in your latest commit.

### Run e2e tests with another database

This approach is not currently used,
neither in the CI/CD nor by core contributors on their local machines.
It is still documented for the sake of completeness:
You can also perform e2e tests using MariaDB/MySQL or PostgreSQL if you want.

Setup a MySQL database inside docker
```
docker run -e "MYSQL_DATABASE=test" -e "MYSQL_ALLOW_EMPTY_PASSWORD=yes" -p 3306:3306 --rm --name mysql mysql:latest #(Ctrl-c to stop the database)
```
Start tests based on the database container
```
TEST_MYSQL_HOST=localhost:3306 TEST_MYSQL_DBNAME=test?multiStatements=true TEST_MYSQL_USERNAME=root TEST_MYSQL_PASSWORD='' make test-e2e-mysql
```

Setup a pgsql database inside docker
```
docker run -e POSTGRES_DB=test -e POSTGRES_PASSWORD=password -p 5432:5432 --rm --name pgsql postgres:latest #(Ctrl-c to stop the database)
```
Start tests based on the database container
```
TEST_PGSQL_HOST=localhost:5432 TEST_PGSQL_DBNAME=test TEST_PGSQL_USERNAME=postgres TEST_PGSQL_PASSWORD=postgres make test-e2e-pgsql
```

### Running individual tests

Example command to run `example.test.e2e.ts` test file:

> **Note**
> Unlike integration tests, this filtering is at the file level, not function

For SQLite:

```
make test-e2e-sqlite#example
```


## Tips and tricks

If you know noteworthy tests that can act as an inspiration for new tests,
please add some details here.

### Understanding and waiting for page loads

[Waiting for a load state](https://playwright.dev/docs/api/class-frame#frame-wait-for-load-state)
sound like a convenient way to ensure the page was loaded,
but it only works once and consecutive calls to it
(e.g. after clicking a button which should reload a page)
return immediately without waiting for *another* load event.

If you match something which is on both the old and the new page,
you might succeed before the page was reloaded,
although the code using a `waitForLoadState` might intuitively suggest
the page was changed before.

Interacting with the page before the reload
(e.g. by opening a dropdown)
might then race and result in flaky tests,
depending on the speed of the hardware running the test.

A possible way to test that an interaction worked is by checking for a known change first.
For example:

- you submit a form and you want to check that the content persisted
- checking for the content directly would succeed even without a page reload
- check for a success message first (will wait until it appears), then verify the content

Alternatively, if you know the backend request that will be made before the reload,
you can explicitly wait for it:

~~~js
const submitted = page.waitForResponse('/my/backend/post/request');
await page.locator('button').first().click(); // perform your interaction
await submitted;
~~~

If the page redirects to another URL,
you can alternatively use:

~~~js
await page.waitForURL('**/target.html');
~~~

### Visual testing

Due to size and frequent updates, we do not host screenshots in the Forgejo repository.
However, it is good practice to ensure that your test is capable of generating relevant and stable screenshots.
Forgejo is regularly tested against visual regressions in a dedicated repository which contains the screenshots:
https://code.forgejo.org/forgejo/visual-browser-testing/

For tests that consume only the `page`,
screenshots are automatically created at the end of each test.

If your test visits different relevant screens or pages during the test,
or creates a custom `page` from context
(e.g. for tests that require a signed-in user)
calling `await save_visual(page);` explicitly in relevant positions is encouraged.

Please confirm locally that your screenshots are stable by performing several runs of your test.
When screenshots are available and reproducible,
check in your test without the screenshots.

When your screenshots differ between runs,
for example because dynamic elements (e.g. timestamps, commit hashes etc)
change between runs,
mask these elements in the `save_visual` function in `utils_e2e.ts`.

#### Working with screenshots

The following environment variables control visual testing:

`VISUAL_TEST=1` will create screenshots in tests/e2e/test-snapshots.
  The test will fail the first time,
  because the screenshots are not included with Forgejo.
  Subsequent runs will comopare against your local copy of the screenshots.

`ACCEPT_VISUAL=1` will overwrite the snapshot images with new images.

### Only sign in if necessary

Signing in takes time and is actually executed step-by-step.
If your test does not rely on a user account, skip this step.

~~~js
test('For anyone', async ({page}) => {
  await page.goto('/somepage');
~~~

If you need a user account, you can use something like:

~~~js
import {test} from './utils_e2e.ts';

// reuse user2 token from scope `shared`
test.use({user: 'user2', authScope: 'shared'})

test('For signed users only', async ({page}) => {

})
~~~

users are created in [utils_e2e_test.go](utils_e2e_test.go)

### Run tests very selectively

Browser testing can take some time.
If you want to iterate fast,
save your time and only run very selected tests.
Use only one browser.

### Skip Safari if it doesn't work

Many contributors have issues getting Safari (webkit)
and especially Safari Mobile to work.

At the top of your test function, you can use:

~~~javascript
test.skip(workerInfo.project.name === 'Mobile Safari', 'Unable to get tests working on Safari Mobile.');
~~~

### Don't forget the formatting.

When writing tests without modifying other frontend code,
it is easy to forget that the JavaScript test files also need formatting.

Run `make lint-frontend-fix`.

### Define new repos

Take a look at `declare_repos_test.go` to see how to add your repositories.
Feel free to improve the logic used there if you need more advanced functionality,
it is a simplified version of the code used in the integration tests.

### Accessibility testing

If you can, perform automated accessibility testing using
[AxeCore](https://github.com/dequelabs/axe-core-npm/blob/develop/packages/playwright/README.md).

Take a look at `shared/forms.ts` and some other places for inspiration.

### List related files coverage

To speed up the CI pipelines and avoid running expensive tests too often,
only a selection of tests is run by default, based on the changed files.

At the top of each playwright test file,
list the files or file patterns that are covered by your test.
Often, these are files that you modified for your feature or bugfix,
or that you looked at (and might still have open in your IDE),
because your fix depends on their behaviour.

#### Which files to watch?

The set of files your test "watches" depends on the kind of test you write.
If you only test for the presence of an element and do no accessibility or placement checks,
you won't detect broken visual appearance and there is little reason to watch CSS files.

However, if your test also checks that an element is correctly positioned
(e.g. that it does not overflow the page),
or has accessibility properties (includes colour contrast),
also list stylesheets that define the behaviour your test depends on.

Watching the place that generate the selectors you use
(typically templates, but can also be JavaScript)
is a must, to ensure that someone modifying the markup notices that your selectors fail
(e.g. because an id or class was renamed).

If you are unsure about the exact set of files, feel free to ask other contributors.

#### How to specify the patterns?

You put filenames and patterns as blocks between two `// @watch` comments.
An example that watches changes on (in order)
a single file,
a full recursive subfolder,
two files with a shorthand pattern,
and a set of files with a certain ending:

~~~
// @watch start
// templates/webhook/shared-settings.tmpl
// templates/repo/settings/**
// web_src/css/{form,repo}.css
// web_src/css/modules/*.css
// @watch end
~~~

The patterns are evaluated on a "first-match" basis.
Under the hood, [gobwas/glob](https://github.com/gobwas/glob) is used.

## Grouped retry for interactions

Sometimes, it can be necessary to retry certain interactions together.
Consider the following procedure:

1. click to open a dropdown
2. interact with content in the dropdown

When for some reason the dropdown does not open,
for example because of it taking time to initialize after page load,
the click will succeed,
but the depending interaction won't,
although playwright repeatedly tries to find the content.

You can [group statements using toPass](https://playwright.dev/docs/test-assertions#expecttopass).
This code retries the dropdown click until the second item is found.

~~~js
await expect(async () => {
  await page.locator('.dropdown').click();
  await page.locator('.dropdown .item').first().click();
}).toPass();
~~~
