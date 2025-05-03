# Forgejo translations

All translations are stored in directories `locale` and `locale_next`.

`locale` is a historical directory that contains translations in INI format. Forgejo inherited it from Gitea, and Gitea inherited it from Gogs.

Because the INI format had many issues and prevented good translatability, in early 2025 Forgejo started switching to a new format - `go-i18n`+`json`.

## Working on base language

Here are some tips:
* when working on non-i18n changes, only change `en-US` files
  * non-base files are normally modified through Weblate. We appreciate the intention to provide localization for the change you're working on, however, modifying those files leads to merge conflicts with Weblate that aren't easy to resolve. [Learn about translating Forgejo](#working-on-other-languages).
* when new strings are added, it's preferred that they're added to `locale_en-US.json`
* when strings are modified in `locale_en-US.ini`, it's preferred that they stay here because moving them around is complicated
* make sure to remove strings if your change renders them unused
* consult https://forgejo.org/docs/next/contributor/localization-english/

### JSON translations

It is preferred that all new strings added to Forgejo UI are added to the JSON translations instead of INI.

Even though Forgejo parser supports nested sections, linters and Weblate do not. Because of this, most strings need to have all sections flattened into their keys like so:
```json
"some.nested.section.key": "UI text"
```

However, plural variations of a string, if it has any, are stored in a nested dictionary:
```json
"some.nested.section.key": {
    "one": "%d comment",
    "other": "%d comments"
}
```

> [!IMPORTANT]
> Please avoid adding unnecessary sections to the keys. Sections like `repo` are vague and represent a large part of the codebase. Keep the sections scoped to where or how the strings are really used, like `user_settings` or `error`.

> [!TIP]
> Due to the flat sections, you can easily find both JSON strings and their usages in the codebase by grepping an entire key.

> [!TIP]
> 3rd party software can determine whether string has plural variations or not from type of it's value in `en-US.json`.

## Working on other languages

Translations are done on Codeberg Translate and not via individual pull requests.

* consult https://forgejo.org/docs/next/contributor/localization/
* see the project: https://translate.codeberg.org/projects/forgejo/
