# Forgejo translations

This directory contains all .INI translations.

## Working on base language

When you work on Forgejo features, you should only modify `locale_en-US.ini`.

* consult https://forgejo.org/docs/next/contributor/localization-english/
* add strings when your change requires doing so
* remove strings when your change renders them unused

## Working on other languages

Translations are done on Codeberg Translate and not via individual pull requests.

* consult https://forgejo.org/docs/next/contributor/localization/
* see the project: https://translate.codeberg.org/projects/forgejo/forgejo/

## Attribution

Forgejo translators are attributed in commit messages and in monthly updates on the website.

Gitea translators are mostly not attributed in this repository because Gitea translation commits are lacking attribution, but it may be preserved on Crowdin.

This directory contains a legacy `TRANSLATORS` file from the Gogs era. It is opt-in and is not actively maintained.
