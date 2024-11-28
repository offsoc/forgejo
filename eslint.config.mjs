import eslintCommunityEslintPluginEslintComments from '@eslint-community/eslint-plugin-eslint-comments';
import stylisticEslintPluginJs from '@stylistic/eslint-plugin-js';
import vitest from '@vitest/eslint-plugin';
import arrayFunc from 'eslint-plugin-array-func';
import eslintPluginImportX from 'eslint-plugin-import-x';
import noJquery from 'eslint-plugin-no-jquery';
import noUseExtendNative from 'eslint-plugin-no-use-extend-native';
import regexp from 'eslint-plugin-regexp';
import sonarjs from 'eslint-plugin-sonarjs';
import unicorn from 'eslint-plugin-unicorn';
import playwright from 'eslint-plugin-playwright';
import vitestGlobals from 'eslint-plugin-vitest-globals';
import wc from 'eslint-plugin-wc';
import globals from 'globals';
import vue from 'eslint-plugin-vue';
import vueScopedCss from 'eslint-plugin-vue-scoped-css';
import tseslint from 'typescript-eslint';

export default tseslint.config(
  ...tseslint.configs.recommended,
  eslintPluginImportX.flatConfigs.typescript,
  {
    ignores: ['web_src/js/vendor', 'web_src/fomantic', 'public/assets/js', 'tests/e2e/reports/'],
  },
  {
    plugins: {
      '@eslint-community/eslint-comments': eslintCommunityEslintPluginEslintComments,
      '@stylistic/js': stylisticEslintPluginJs,
      '@vitest': vitest,
      'array-func': arrayFunc,
      'import-x': eslintPluginImportX,
      'no-jquery': noJquery,
      'no-use-extend-native': noUseExtendNative,
      regexp,
      sonarjs,
      unicorn,
      playwright,
      'vitest-globals': vitestGlobals,
      vue,
      'vue-scoped-css': vueScopedCss,
      wc,
    },

    linterOptions: {
      reportUnusedDisableDirectives: true,
    },

    languageOptions: {
      globals: {
        ...globals.node,
      },
      parserOptions: {
        ecmaVersion: 'latest',
      },

      ecmaVersion: 'latest',
      sourceType: 'module',
    },
    rules: {
      '@typescript-eslint/no-unused-vars': 'off', // TODO: enable this rule again

      '@eslint-community/eslint-comments/disable-enable-pair': [2],
      '@eslint-community/eslint-comments/no-aggregating-enable': [2],
      '@eslint-community/eslint-comments/no-duplicate-disable': [2],
      '@eslint-community/eslint-comments/no-restricted-disable': [0],
      '@eslint-community/eslint-comments/no-unlimited-disable': [2],
      '@eslint-community/eslint-comments/no-unused-disable': [2],
      '@eslint-community/eslint-comments/no-unused-enable': [2],
      '@eslint-community/eslint-comments/no-use': [0],
      '@eslint-community/eslint-comments/require-description': [0],
      '@stylistic/js/array-bracket-newline': [0],
      '@stylistic/js/array-bracket-spacing': [2, 'never'],
      '@stylistic/js/array-element-newline': [0],
      '@stylistic/js/arrow-parens': [2, 'always'],

      '@stylistic/js/arrow-spacing': [2, {
        before: true,
        after: true,
      }],

      '@stylistic/js/block-spacing': [0],

      '@stylistic/js/brace-style': [2, '1tbs', {
        allowSingleLine: true,
      }],

      '@stylistic/js/comma-dangle': [2, 'always-multiline'],

      '@stylistic/js/comma-spacing': [2, {
        before: false,
        after: true,
      }],

      '@stylistic/js/comma-style': [2, 'last'],
      '@stylistic/js/computed-property-spacing': [2, 'never'],
      '@stylistic/js/dot-location': [2, 'property'],
      '@stylistic/js/eol-last': [2],
      '@stylistic/js/function-call-spacing': [2, 'never'],
      '@stylistic/js/function-call-argument-newline': [0],
      '@stylistic/js/function-paren-newline': [0],
      '@stylistic/js/generator-star-spacing': [0],
      '@stylistic/js/implicit-arrow-linebreak': [0],

      '@stylistic/js/indent': [2, 2, {
        ignoreComments: true,
        SwitchCase: 1,
      }],

      '@stylistic/js/key-spacing': [2],
      '@stylistic/js/keyword-spacing': [2],
      '@stylistic/js/linebreak-style': [2, 'unix'],
      '@stylistic/js/lines-around-comment': [0],
      '@stylistic/js/lines-between-class-members': [0],
      '@stylistic/js/max-len': [0],
      '@stylistic/js/max-statements-per-line': [0],
      '@stylistic/js/multiline-ternary': [0],
      '@stylistic/js/new-parens': [2],
      '@stylistic/js/newline-per-chained-call': [0],
      '@stylistic/js/no-confusing-arrow': [0],
      '@stylistic/js/no-extra-parens': [0],
      '@stylistic/js/no-extra-semi': [2],
      '@stylistic/js/no-floating-decimal': [0],
      '@stylistic/js/no-mixed-operators': [0],
      '@stylistic/js/no-mixed-spaces-and-tabs': [2],

      '@stylistic/js/no-multi-spaces': [2, {
        ignoreEOLComments: true,

        exceptions: {
          Property: true,
        },
      }],

      '@stylistic/js/no-multiple-empty-lines': [2, {
        max: 1,
        maxEOF: 0,
        maxBOF: 0,
      }],

      '@stylistic/js/no-tabs': [2],
      '@stylistic/js/no-trailing-spaces': [2],
      '@stylistic/js/no-whitespace-before-property': [2],
      '@stylistic/js/nonblock-statement-body-position': [2],
      '@stylistic/js/object-curly-newline': [0],
      '@stylistic/js/object-curly-spacing': [2, 'never'],
      '@stylistic/js/object-property-newline': [0],
      '@stylistic/js/one-var-declaration-per-line': [0],
      '@stylistic/js/operator-linebreak': [2, 'after'],
      '@stylistic/js/padded-blocks': [2, 'never'],
      '@stylistic/js/padding-line-between-statements': [0],
      '@stylistic/js/quote-props': [0],

      '@stylistic/js/quotes': [2, 'single', {
        avoidEscape: true,
        allowTemplateLiterals: true,
      }],

      '@stylistic/js/rest-spread-spacing': [2, 'never'],

      '@stylistic/js/semi': [2, 'always', {
        omitLastInOneLineBlock: true,
      }],

      '@stylistic/js/semi-spacing': [2, {
        before: false,
        after: true,
      }],

      '@stylistic/js/semi-style': [2, 'last'],
      '@stylistic/js/space-before-blocks': [2, 'always'],

      '@stylistic/js/space-before-function-paren': [2, {
        anonymous: 'ignore',
        named: 'never',
        asyncArrow: 'always',
      }],

      '@stylistic/js/space-in-parens': [2, 'never'],
      '@stylistic/js/space-infix-ops': [2],
      '@stylistic/js/space-unary-ops': [2],
      '@stylistic/js/spaced-comment': [2, 'always'],
      '@stylistic/js/switch-colon-spacing': [2],
      '@stylistic/js/template-curly-spacing': [2, 'never'],
      '@stylistic/js/template-tag-spacing': [2, 'never'],
      '@stylistic/js/wrap-iife': [2, 'inside'],
      '@stylistic/js/wrap-regex': [0],
      '@stylistic/js/yield-star-spacing': [2, 'after'],
      'accessor-pairs': [2],

      'array-callback-return': [2, {
        checkForEach: true,
      }],

      'array-func/avoid-reverse': [2],
      'array-func/from-map': [2],
      'array-func/no-unnecessary-this-arg': [2],
      'array-func/prefer-array-from': [2],
      'array-func/prefer-flat-map': [0],
      'array-func/prefer-flat': [0],
      'arrow-body-style': [0],
      'block-scoped-var': [2],
      camelcase: [0],
      'capitalized-comments': [0],
      'class-methods-use-this': [0],
      complexity: [0],
      'consistent-return': [0],
      'consistent-this': [0],
      'constructor-super': [2],
      curly: [0],
      'default-case-last': [2],
      'default-case': [0],
      'default-param-last': [0],
      'dot-notation': [0],
      eqeqeq: [2],
      'for-direction': [2],
      'func-name-matching': [2],
      'func-names': [0],
      'func-style': [0],
      'getter-return': [2],
      'grouped-accessor-pairs': [2],
      'guard-for-in': [0],
      'id-blacklist': [0],
      'id-length': [0],
      'id-match': [0],
      'init-declarations': [0],
      'line-comment-position': [0],
      'logical-assignment-operators': [0],
      'max-classes-per-file': [0],
      'max-depth': [0],
      'max-lines-per-function': [0],
      'max-lines': [0],
      'max-nested-callbacks': [0],
      'max-params': [0],
      'max-statements': [0],
      'multiline-comment-style': [2, 'separate-lines'],
      'new-cap': [0],
      'no-alert': [0],
      'no-array-constructor': [2],
      'no-async-promise-executor': [0],
      'no-await-in-loop': [0],
      'no-bitwise': [0],
      'no-buffer-constructor': [0],
      'no-caller': [2],
      'no-case-declarations': [2],
      'no-class-assign': [2],
      'no-compare-neg-zero': [2],
      'no-cond-assign': [2, 'except-parens'],

      'no-console': [1, {
        allow: ['debug', 'info', 'warn', 'error'],
      }],

      'no-const-assign': [2],
      'no-constant-binary-expression': [2],
      'no-constant-condition': [0],
      'no-constructor-return': [2],
      'no-continue': [0],
      'no-control-regex': [0],
      'no-debugger': [1],
      'no-delete-var': [2],
      'no-div-regex': [0],
      'no-dupe-args': [2],
      'no-dupe-class-members': [2],
      'no-dupe-else-if': [2],
      'no-dupe-keys': [2],
      'no-duplicate-case': [2],
      'no-duplicate-imports': [2],
      'no-else-return': [2],
      'no-empty-character-class': [2],
      'no-empty-function': [0],
      'no-empty-pattern': [2],
      'no-empty-static-block': [2],

      'no-empty': [2, {
        allowEmptyCatch: true,
      }],

      'no-eq-null': [2],
      'no-eval': [2],
      'no-ex-assign': [2],
      'no-extend-native': [2],
      'no-extra-bind': [2],
      'no-extra-boolean-cast': [2],
      'no-extra-label': [0],
      'no-fallthrough': [2],
      'no-func-assign': [2],
      'no-global-assign': [2],
      'no-implicit-coercion': [2],
      'no-implicit-globals': [0],
      'no-implied-eval': [2],
      'no-import-assign': [2],
      'no-inline-comments': [0],
      'no-inner-declarations': [2],
      'no-invalid-regexp': [2],
      'no-invalid-this': [0],
      'no-irregular-whitespace': [2],
      'no-iterator': [2],
      'no-jquery/no-ajax-events': [2],
      'no-jquery/no-ajax': [2],
      'no-jquery/no-and-self': [2],
      'no-jquery/no-animate-toggle': [2],
      'no-jquery/no-animate': [2],
      'no-jquery/no-append-html': [2],
      'no-jquery/no-attr': [2],
      'no-jquery/no-bind': [2],
      'no-jquery/no-box-model': [2],
      'no-jquery/no-browser': [2],
      'no-jquery/no-camel-case': [2],
      'no-jquery/no-class-state': [2],
      'no-jquery/no-class': [0],
      'no-jquery/no-clone': [2],
      'no-jquery/no-closest': [0],
      'no-jquery/no-constructor-attributes': [2],
      'no-jquery/no-contains': [2],
      'no-jquery/no-context-prop': [2],
      'no-jquery/no-css': [2],
      'no-jquery/no-data': [0],
      'no-jquery/no-deferred': [2],
      'no-jquery/no-delegate': [2],
      'no-jquery/no-each-collection': [0],
      'no-jquery/no-each-util': [0],
      'no-jquery/no-each': [0],
      'no-jquery/no-error-shorthand': [2],
      'no-jquery/no-error': [2],
      'no-jquery/no-escape-selector': [2],
      'no-jquery/no-event-shorthand': [2],
      'no-jquery/no-extend': [2],
      'no-jquery/no-fade': [2],
      'no-jquery/no-filter': [0],
      'no-jquery/no-find-collection': [0],
      'no-jquery/no-find-util': [2],
      'no-jquery/no-find': [0],
      'no-jquery/no-fx-interval': [2],
      'no-jquery/no-global-eval': [2],
      'no-jquery/no-global-selector': [0],
      'no-jquery/no-grep': [2],
      'no-jquery/no-has': [2],
      'no-jquery/no-hold-ready': [2],
      'no-jquery/no-html': [0],
      'no-jquery/no-in-array': [2],
      'no-jquery/no-is-array': [2],
      'no-jquery/no-is-empty-object': [2],
      'no-jquery/no-is-function': [2],
      'no-jquery/no-is-numeric': [2],
      'no-jquery/no-is-plain-object': [2],
      'no-jquery/no-is-window': [2],
      'no-jquery/no-is': [2],
      'no-jquery/no-jquery-constructor': [0],
      'no-jquery/no-live': [2],
      'no-jquery/no-load-shorthand': [2],
      'no-jquery/no-load': [2],
      'no-jquery/no-map-collection': [0],
      'no-jquery/no-map-util': [2],
      'no-jquery/no-map': [2],
      'no-jquery/no-merge': [2],
      'no-jquery/no-node-name': [2],
      'no-jquery/no-noop': [2],
      'no-jquery/no-now': [2],
      'no-jquery/no-on-ready': [2],
      'no-jquery/no-other-methods': [0],
      'no-jquery/no-other-utils': [2],
      'no-jquery/no-param': [2],
      'no-jquery/no-parent': [0],
      'no-jquery/no-parents': [2],
      'no-jquery/no-parse-html-literal': [2],
      'no-jquery/no-parse-html': [2],
      'no-jquery/no-parse-json': [2],
      'no-jquery/no-parse-xml': [2],
      'no-jquery/no-prop': [2],
      'no-jquery/no-proxy': [2],
      'no-jquery/no-ready-shorthand': [2],
      'no-jquery/no-ready': [2],
      'no-jquery/no-selector-prop': [2],
      'no-jquery/no-serialize': [2],
      'no-jquery/no-size': [2],
      'no-jquery/no-sizzle': [0],
      'no-jquery/no-slide': [2],
      'no-jquery/no-sub': [2],
      'no-jquery/no-support': [2],
      'no-jquery/no-text': [0],
      'no-jquery/no-trigger': [0],
      'no-jquery/no-trim': [2],
      'no-jquery/no-type': [2],
      'no-jquery/no-unique': [2],
      'no-jquery/no-unload-shorthand': [2],
      'no-jquery/no-val': [0],
      'no-jquery/no-visibility': [2],
      'no-jquery/no-when': [2],
      'no-jquery/no-wrap': [2],
      'no-jquery/variable-pattern': [2],
      'no-label-var': [2],
      'no-labels': [0],
      'no-lone-blocks': [2],
      'no-lonely-if': [0],
      'no-loop-func': [0],
      'no-loss-of-precision': [2],
      'no-magic-numbers': [0],
      'no-misleading-character-class': [2],
      'no-multi-assign': [0],
      'no-multi-str': [2],
      'no-negated-condition': [0],
      'no-nested-ternary': [0],
      'no-new-func': [2],
      'no-new-native-nonconstructor': [2],
      'no-new-object': [2],
      'no-new-symbol': [2],
      'no-new-wrappers': [2],
      'no-new': [0],
      'no-nonoctal-decimal-escape': [2],
      'no-obj-calls': [2],
      'no-octal-escape': [2],
      'no-octal': [2],
      'no-param-reassign': [0],
      'no-plusplus': [0],
      'no-promise-executor-return': [0],
      'no-proto': [2],
      'no-prototype-builtins': [2],
      'no-redeclare': [2],
      'no-regex-spaces': [2],
      'no-restricted-exports': [0],

      'no-restricted-globals': [
        2,
        'addEventListener',
        'blur',
        'close',
        'closed',
        'confirm',
        'defaultStatus',
        'defaultstatus',
        'error',
        'event',
        'external',
        'find',
        'focus',
        'frameElement',
        'frames',
        'history',
        'innerHeight',
        'innerWidth',
        'isFinite',
        'isNaN',
        'length',
        'location',
        'locationbar',
        'menubar',
        'moveBy',
        'moveTo',
        'name',
        'onblur',
        'onerror',
        'onfocus',
        'onload',
        'onresize',
        'onunload',
        'open',
        'opener',
        'opera',
        'outerHeight',
        'outerWidth',
        'pageXOffset',
        'pageYOffset',
        'parent',
        'print',
        'removeEventListener',
        'resizeBy',
        'resizeTo',
        'screen',
        'screenLeft',
        'screenTop',
        'screenX',
        'screenY',
        'scroll',
        'scrollbars',
        'scrollBy',
        'scrollTo',
        'scrollX',
        'scrollY',
        'self',
        'status',
        'statusbar',
        'stop',
        'toolbar',
        'top',
        '__dirname',
        '__filename',
      ],

      'no-restricted-imports': [0],

      'no-restricted-syntax': [
        2,
        'WithStatement',
        'ForInStatement',
        'LabeledStatement',
        'SequenceExpression',
        {
          selector: "CallExpression[callee.name='fetch']",
          message: 'use modules/fetch.js instead',
        },
      ],

      'no-return-assign': [0],
      'no-script-url': [2],

      'no-self-assign': [2, {
        props: true,
      }],

      'no-self-compare': [2],
      'no-sequences': [2],
      'no-setter-return': [2],
      'no-shadow-restricted-names': [2],
      'no-shadow': [0],
      'no-sparse-arrays': [2],
      'no-template-curly-in-string': [2],
      'no-ternary': [0],
      'no-this-before-super': [2],
      'no-throw-literal': [2],
      'no-undef-init': [2],

      'no-undef': [2, {
        typeof: true,
      }],

      'no-undefined': [0],
      'no-underscore-dangle': [0],
      'no-unexpected-multiline': [2],
      'no-unmodified-loop-condition': [2],
      'no-unneeded-ternary': [2],
      'no-unreachable-loop': [2],
      'no-unreachable': [2],
      'no-unsafe-finally': [2],
      'no-unsafe-negation': [2],
      'no-unused-expressions': [2],
      'no-unused-labels': [2],
      'no-unused-private-class-members': [2],

      'no-unused-vars': [2, {
        args: 'all',
        argsIgnorePattern: '^_',
        varsIgnorePattern: '^_',
        caughtErrorsIgnorePattern: '^_',
        destructuredArrayIgnorePattern: '^_',
        ignoreRestSiblings: false,
      }],

      'no-use-before-define': [2, {
        functions: false,
        classes: true,
        variables: true,
        allowNamedExports: true,
      }],

      'no-use-extend-native/no-use-extend-native': [2],
      'no-useless-backreference': [2],
      'no-useless-call': [2],
      'no-useless-catch': [2],
      'no-useless-computed-key': [2],
      'no-useless-concat': [2],
      'no-useless-constructor': [2],
      'no-useless-escape': [2],
      'no-useless-rename': [2],
      'no-useless-return': [2],
      'no-var': [2],
      'no-void': [2],
      'no-warning-comments': [0],
      'no-with': [0],
      'object-shorthand': [2, 'always'],
      'one-var-declaration-per-line': [0],
      'one-var': [0],
      'operator-assignment': [2, 'always'],
      'operator-linebreak': [2, 'after'],

      'prefer-arrow-callback': [2, {
        allowNamedFunctions: true,
        allowUnboundThis: true,
      }],

      'prefer-const': [2, {
        destructuring: 'all',
        ignoreReadBeforeAssign: true,
      }],

      'prefer-destructuring': [0],
      'prefer-exponentiation-operator': [2],
      'prefer-named-capture-group': [0],
      'prefer-numeric-literals': [2],
      'prefer-object-has-own': [2],
      'prefer-object-spread': [2],

      'prefer-promise-reject-errors': [2, {
        allowEmptyReject: false,
      }],

      'prefer-regex-literals': [2],
      'prefer-rest-params': [2],
      'prefer-spread': [2],
      'prefer-template': [2],
      radix: [2, 'as-needed'],
      'regexp/confusing-quantifier': [2],
      'regexp/control-character-escape': [2],
      'regexp/hexadecimal-escape': [0],
      'regexp/letter-case': [0],
      'regexp/match-any': [2],
      'regexp/negation': [2],
      'regexp/no-contradiction-with-assertion': [0],
      'regexp/no-control-character': [0],
      'regexp/no-dupe-characters-character-class': [2],
      'regexp/no-dupe-disjunctions': [2],
      'regexp/no-empty-alternative': [2],
      'regexp/no-empty-capturing-group': [2],
      'regexp/no-empty-character-class': [0],
      'regexp/no-empty-group': [2],
      'regexp/no-empty-lookarounds-assertion': [2],
      'regexp/no-empty-string-literal': [2],
      'regexp/no-escape-backspace': [2],
      'regexp/no-extra-lookaround-assertions': [0],
      'regexp/no-invalid-regexp': [2],
      'regexp/no-invisible-character': [2],
      'regexp/no-lazy-ends': [2],
      'regexp/no-legacy-features': [2],
      'regexp/no-misleading-capturing-group': [0],
      'regexp/no-misleading-unicode-character': [0],
      'regexp/no-missing-g-flag': [2],
      'regexp/no-non-standard-flag': [2],
      'regexp/no-obscure-range': [2],
      'regexp/no-octal': [2],
      'regexp/no-optional-assertion': [2],
      'regexp/no-potentially-useless-backreference': [2],
      'regexp/no-standalone-backslash': [2],
      'regexp/no-super-linear-backtracking': [0],
      'regexp/no-super-linear-move': [0],
      'regexp/no-trivially-nested-assertion': [2],
      'regexp/no-trivially-nested-quantifier': [2],
      'regexp/no-unused-capturing-group': [0],
      'regexp/no-useless-assertions': [2],
      'regexp/no-useless-backreference': [2],
      'regexp/no-useless-character-class': [2],
      'regexp/no-useless-dollar-replacements': [2],
      'regexp/no-useless-escape': [2],
      'regexp/no-useless-flag': [2],
      'regexp/no-useless-lazy': [2],
      'regexp/no-useless-non-capturing-group': [2],
      'regexp/no-useless-quantifier': [2],
      'regexp/no-useless-range': [2],
      'regexp/no-useless-set-operand': [2],
      'regexp/no-useless-string-literal': [2],
      'regexp/no-useless-two-nums-quantifier': [2],
      'regexp/no-zero-quantifier': [2],
      'regexp/optimal-lookaround-quantifier': [2],
      'regexp/optimal-quantifier-concatenation': [0],
      'regexp/prefer-character-class': [0],
      'regexp/prefer-d': [0],
      'regexp/prefer-escape-replacement-dollar-char': [0],
      'regexp/prefer-lookaround': [0],
      'regexp/prefer-named-backreference': [0],
      'regexp/prefer-named-capture-group': [0],
      'regexp/prefer-named-replacement': [0],
      'regexp/prefer-plus-quantifier': [2],
      'regexp/prefer-predefined-assertion': [2],
      'regexp/prefer-quantifier': [0],
      'regexp/prefer-question-quantifier': [2],
      'regexp/prefer-range': [2],
      'regexp/prefer-regexp-exec': [2],
      'regexp/prefer-regexp-test': [2],
      'regexp/prefer-result-array-groups': [0],
      'regexp/prefer-set-operation': [2],
      'regexp/prefer-star-quantifier': [2],
      'regexp/prefer-unicode-codepoint-escapes': [2],
      'regexp/prefer-w': [0],
      'regexp/require-unicode-regexp': [0],
      'regexp/simplify-set-operations': [2],
      'regexp/sort-alternatives': [0],
      'regexp/sort-character-class-elements': [0],
      'regexp/sort-flags': [0],
      'regexp/strict': [2],
      'regexp/unicode-escape': [0],
      'regexp/use-ignore-case': [0],
      'require-atomic-updates': [0],
      'require-await': [0],
      'require-unicode-regexp': [0],
      'require-yield': [2],
      'sonarjs/cognitive-complexity': [0],
      'sonarjs/elseif-without-else': [0],
      'sonarjs/max-switch-cases': [0],
      'sonarjs/no-all-duplicated-branches': [2],
      'sonarjs/no-collapsible-if': [0],
      'sonarjs/no-collection-size-mischeck': [2],
      'sonarjs/no-duplicate-string': [0],
      'sonarjs/no-duplicated-branches': [0],
      'sonarjs/no-element-overwrite': [2],
      'sonarjs/no-empty-collection': [2],
      'sonarjs/no-extra-arguments': [2],
      'sonarjs/no-gratuitous-expressions': [2],
      'sonarjs/no-identical-conditions': [2],
      'sonarjs/no-identical-expressions': [2],
      'sonarjs/no-identical-functions': [2, 5],
      'sonarjs/no-ignored-return': [2],
      'sonarjs/no-inverted-boolean-check': [2],
      'sonarjs/no-nested-switch': [0],
      'sonarjs/no-nested-template-literals': [0],
      'sonarjs/no-one-iteration-loop': [2],
      'sonarjs/no-redundant-boolean': [2],
      'sonarjs/no-redundant-jump': [2],
      'sonarjs/no-same-line-conditional': [2],
      'sonarjs/no-small-switch': [0],
      'sonarjs/no-unused-collection': [2],
      'sonarjs/no-use-of-empty-return-value': [2],
      'sonarjs/no-useless-catch': [2],
      'sonarjs/non-existent-operator': [2],
      'sonarjs/prefer-immediate-return': [0],
      'sonarjs/prefer-object-literal': [0],
      'sonarjs/prefer-single-boolean-return': [0],
      'sonarjs/prefer-while': [2],
      'sort-imports': [0],
      'sort-keys': [0],
      'sort-vars': [0],
      strict: [0],
      'symbol-description': [2],
      'unicode-bom': [2, 'never'],
      'unicorn/better-regex': [0],
      'unicorn/catch-error-name': [0],
      'unicorn/consistent-destructuring': [2],
      'unicorn/consistent-empty-array-spread': [2],
      'unicorn/consistent-existence-index-check': [2],
      'unicorn/consistent-function-scoping': [2],
      'unicorn/custom-error-definition': [0],
      'unicorn/empty-brace-spaces': [2],
      'unicorn/error-message': [0],
      'unicorn/escape-case': [0],
      'unicorn/expiring-todo-comments': [0],
      'unicorn/explicit-length-check': [0],
      'unicorn/filename-case': [0],
      'unicorn/import-index': [0],
      'unicorn/import-style': [0],
      'unicorn/new-for-builtins': [2],
      'unicorn/no-abusive-eslint-disable': [0],
      'unicorn/no-anonymous-default-export': [0],
      'unicorn/no-array-callback-reference': [0],
      'unicorn/no-array-for-each': [2],
      'unicorn/no-array-method-this-argument': [2],
      'unicorn/no-array-push-push': [2],
      'unicorn/no-array-reduce': [2],
      'unicorn/no-await-expression-member': [0],
      'unicorn/no-await-in-promise-methods': [2],
      'unicorn/no-console-spaces': [0],
      'unicorn/no-document-cookie': [2],
      'unicorn/no-empty-file': [2],
      'unicorn/no-for-loop': [0],
      'unicorn/no-hex-escape': [0],
      'unicorn/no-instanceof-array': [0],
      'unicorn/no-invalid-fetch-options': [2],
      'unicorn/no-invalid-remove-event-listener': [2],
      'unicorn/no-keyword-prefix': [0],
      'unicorn/no-length-as-slice-end': [2],
      'unicorn/no-lonely-if': [2],
      'unicorn/no-magic-array-flat-depth': [0],
      'unicorn/no-negated-condition': [0],
      'unicorn/no-negation-in-equality-check': [2],
      'unicorn/no-nested-ternary': [0],
      'unicorn/no-new-array': [0],
      'unicorn/no-new-buffer': [0],
      'unicorn/no-null': [0],
      'unicorn/no-object-as-default-parameter': [0],
      'unicorn/no-process-exit': [0],
      'unicorn/no-single-promise-in-promise-methods': [2],
      'unicorn/no-static-only-class': [2],
      'unicorn/no-thenable': [2],
      'unicorn/no-this-assignment': [2],
      'unicorn/no-typeof-undefined': [2],
      'unicorn/no-unnecessary-await': [2],
      'unicorn/no-unnecessary-polyfills': [2],
      'unicorn/no-unreadable-array-destructuring': [0],
      'unicorn/no-unreadable-iife': [2],
      'unicorn/no-unused-properties': [2],
      'unicorn/no-useless-fallback-in-spread': [2],
      'unicorn/no-useless-length-check': [2],
      'unicorn/no-useless-promise-resolve-reject': [2],
      'unicorn/no-useless-spread': [2],
      'unicorn/no-useless-switch-case': [2],
      'unicorn/no-useless-undefined': [0],
      'unicorn/no-zero-fractions': [2],
      'unicorn/number-literal-case': [0],
      'unicorn/numeric-separators-style': [0],
      'unicorn/prefer-add-event-listener': [2],
      'unicorn/prefer-array-find': [2],
      'unicorn/prefer-array-flat-map': [2],
      'unicorn/prefer-array-flat': [2],
      'unicorn/prefer-array-index-of': [2],
      'unicorn/prefer-array-some': [2],
      'unicorn/prefer-at': [0],
      'unicorn/prefer-blob-reading-methods': [2],
      'unicorn/prefer-code-point': [0],
      'unicorn/prefer-date-now': [2],
      'unicorn/prefer-default-parameters': [0],
      'unicorn/prefer-dom-node-append': [2],
      'unicorn/prefer-dom-node-dataset': [0],
      'unicorn/prefer-dom-node-remove': [2],
      'unicorn/prefer-dom-node-text-content': [2],
      'unicorn/prefer-event-target': [2],
      'unicorn/prefer-export-from': [0],
      'unicorn/prefer-global-this': [0],
      'unicorn/prefer-includes': [2],
      'unicorn/prefer-json-parse-buffer': [0],
      'unicorn/prefer-keyboard-event-key': [2],
      'unicorn/prefer-logical-operator-over-ternary': [2],
      'unicorn/prefer-math-min-max': [2],
      'unicorn/prefer-math-trunc': [2],
      'unicorn/prefer-modern-dom-apis': [0],
      'unicorn/prefer-modern-math-apis': [2],
      'unicorn/prefer-module': [2],
      'unicorn/prefer-native-coercion-functions': [2],
      'unicorn/prefer-negative-index': [2],
      'unicorn/prefer-node-protocol': [2],
      'unicorn/prefer-number-properties': [0],
      'unicorn/prefer-object-from-entries': [2],
      'unicorn/prefer-object-has-own': [0],
      'unicorn/prefer-optional-catch-binding': [2],
      'unicorn/prefer-prototype-methods': [0],
      'unicorn/prefer-query-selector': [0],
      'unicorn/prefer-reflect-apply': [0],
      'unicorn/prefer-regexp-test': [2],
      'unicorn/prefer-set-has': [0],
      'unicorn/prefer-set-size': [2],
      'unicorn/prefer-spread': [0],
      'unicorn/prefer-string-raw': [0],
      'unicorn/prefer-string-replace-all': [0],
      'unicorn/prefer-string-slice': [0],
      'unicorn/prefer-string-starts-ends-with': [2],
      'unicorn/prefer-string-trim-start-end': [2],
      'unicorn/prefer-structured-clone': [2],
      'unicorn/prefer-switch': [0],
      'unicorn/prefer-ternary': [0],
      'unicorn/prefer-text-content': [2],
      'unicorn/prefer-top-level-await': [0],
      'unicorn/prefer-type-error': [0],
      'unicorn/prevent-abbreviations': [0],
      'unicorn/relative-url-style': [2],
      'unicorn/require-array-join-separator': [2],
      'unicorn/require-number-to-fixed-digits-argument': [2],
      'unicorn/require-post-message-target-origin': [0],
      'unicorn/string-content': [0],
      'unicorn/switch-case-braces': [0],
      'unicorn/template-indent': [2],
      'unicorn/text-encoding-identifier-case': [0],
      'unicorn/throw-new-error': [2],
      'use-isnan': [2],

      'valid-typeof': [2, {
        requireStringLiterals: true,
      }],

      'vars-on-top': [0],
      'wc/attach-shadow-constructor': [2],
      'wc/define-tag-after-class-definition': [0],
      'wc/expose-class-on-global': [0],
      'wc/file-name-matches-element': [2],
      'wc/guard-define-call': [0],
      'wc/guard-super-call': [2],
      'wc/max-elements-per-file': [0],
      'wc/no-child-traversal-in-attributechangedcallback': [2],
      'wc/no-child-traversal-in-connectedcallback': [2],
      'wc/no-closed-shadow-root': [2],
      'wc/no-constructor-attributes': [2],
      'wc/no-constructor-params': [2],
      'wc/no-constructor': [2],
      'wc/no-customized-built-in-elements': [2],
      'wc/no-exports-with-element': [0],
      'wc/no-invalid-element-name': [2],
      'wc/no-invalid-extends': [2],
      'wc/no-method-prefixed-with-on': [2],
      'wc/no-self-class': [2],
      'wc/no-typos': [2],
      'wc/require-listener-teardown': [2],
      'wc/tag-name-matches-class': [2],
      yoda: [2, 'never'],
    },
  },
  {
    ignores: ['*.vue', '**/*.vue'],
    rules: {
      'import-x/consistent-type-specifier-style': [0],
      'import-x/default': [0],
      'import-x/dynamic-import-chunkname': [0],
      'import-x/export': [2],
      'import-x/exports-last': [0],

      'import-x/extensions': [2, 'always', {
        ignorePackages: true,
      }],

      'import-x/first': [2],
      'import-x/group-exports': [0],
      'import-x/max-dependencies': [0],
      'import-x/named': [2],
      'import-x/namespace': [0],
      'import-x/newline-after-import': [0],
      'import-x/no-absolute-path': [0],
      'import-x/no-amd': [2],
      'import-x/no-anonymous-default-export': [0],
      'import-x/no-commonjs': [2],

      'import-x/no-cycle': [2, {
        ignoreExternal: true,
        maxDepth: 1,
      }],

      'import-x/no-default-export': [0],
      'import-x/no-deprecated': [0],
      'import-x/no-dynamic-require': [0],
      'import-x/no-empty-named-blocks': [2],
      'import-x/no-extraneous-dependencies': [2],
      'import-x/no-import-module-exports': [0],
      'import-x/no-internal-modules': [0],
      'import-x/no-mutable-exports': [0],
      'import-x/no-named-as-default-member': [0],
      'import-x/no-named-as-default': [2],
      'import-x/no-named-default': [0],
      'import-x/no-named-export': [0],
      'import-x/no-namespace': [0],
      'import-x/no-nodejs-modules': [0],
      'import-x/no-relative-packages': [0],
      'import-x/no-relative-parent-imports': [0],
      'import-x/no-restricted-paths': [0],
      'import-x/no-self-import': [2],
      'import-x/no-unassigned-import': [0],

      'import-x/no-unresolved': [2, {
        commonjs: true,
        ignore: ['\\?.+$', '^vitest/'],
      }],

      'import-x/no-useless-path-segments': [2, {
        commonjs: true,
      }],

      'import-x/no-webpack-loader-syntax': [2],
      'import-x/order': [0],
      'import-x/prefer-default-export': [0],
      'import-x/unambiguous': [0],
    },
  },
  {
    files: ['web_src/**/*'],
    languageOptions: {
      globals: {
        __webpack_public_path__: true,
        process: false,
      },
    },
  }, {
    files: ['web_src/**/*', 'docs/**/*'],

    languageOptions: {
      globals: {
        ...globals.browser,
      },
    },
  }, {
    files: ['web_src/**/*worker.*'],

    languageOptions: {
      globals: {
        ...globals.worker,
      },
    },

    rules: {
      'no-restricted-globals': [
        2,
        'addEventListener',
        'blur',
        'close',
        'closed',
        'confirm',
        'defaultStatus',
        'defaultstatus',
        'error',
        'event',
        'external',
        'find',
        'focus',
        'frameElement',
        'frames',
        'history',
        'innerHeight',
        'innerWidth',
        'isFinite',
        'isNaN',
        'length',
        'locationbar',
        'menubar',
        'moveBy',
        'moveTo',
        'name',
        'onblur',
        'onerror',
        'onfocus',
        'onload',
        'onresize',
        'onunload',
        'open',
        'opener',
        'opera',
        'outerHeight',
        'outerWidth',
        'pageXOffset',
        'pageYOffset',
        'parent',
        'print',
        'removeEventListener',
        'resizeBy',
        'resizeTo',
        'screen',
        'screenLeft',
        'screenTop',
        'screenX',
        'screenY',
        'scroll',
        'scrollbars',
        'scrollBy',
        'scrollTo',
        'scrollX',
        'scrollY',
        'status',
        'statusbar',
        'stop',
        'toolbar',
        'top',
      ],
    },
  }, {
    files: ['**/*.config.*'],
    languageOptions: {
      ecmaVersion: 'latest',
    },
    rules: {
      'import-x/no-unused-modules': [0],
      'import-x/no-unresolved': [0],
      'import-x/no-named-as-default': [0],
    },
  }, {
    files: ['**/*.test.*', 'web_src/js/test/setup.js'],
    languageOptions: {
      globals: {
        ...vitestGlobals.environments.env.globals,
      },
    },

    rules: {
      '@vitest/consistent-test-filename': [0],
      '@vitest/consistent-test-it': [0],
      '@vitest/expect-expect': [0],
      '@vitest/max-expects': [0],
      '@vitest/max-nested-describe': [0],
      '@vitest/no-alias-methods': [0],
      '@vitest/no-commented-out-tests': [0],
      '@vitest/no-conditional-expect': [0],
      '@vitest/no-conditional-in-test': [0],
      '@vitest/no-conditional-tests': [0],
      '@vitest/no-disabled-tests': [0],
      '@vitest/no-done-callback': [0],
      '@vitest/no-duplicate-hooks': [0],
      '@vitest/no-focused-tests': [0],
      '@vitest/no-hooks': [0],
      '@vitest/no-identical-title': [2],
      '@vitest/no-interpolation-in-snapshots': [0],
      '@vitest/no-large-snapshots': [0],
      '@vitest/no-mocks-import': [0],
      '@vitest/no-restricted-matchers': [0],
      '@vitest/no-restricted-vi-methods': [0],
      '@vitest/no-standalone-expect': [0],
      '@vitest/no-test-prefixes': [0],
      '@vitest/no-test-return-statement': [0],
      '@vitest/prefer-called-with': [0],
      '@vitest/prefer-comparison-matcher': [0],
      '@vitest/prefer-each': [0],
      '@vitest/prefer-equality-matcher': [0],
      '@vitest/prefer-expect-resolves': [0],
      '@vitest/prefer-hooks-in-order': [0],
      '@vitest/prefer-hooks-on-top': [2],
      '@vitest/prefer-lowercase-title': [0],
      '@vitest/prefer-mock-promise-shorthand': [0],
      '@vitest/prefer-snapshot-hint': [0],
      '@vitest/prefer-spy-on': [0],
      '@vitest/prefer-strict-equal': [0],
      '@vitest/prefer-to-be': [0],
      '@vitest/prefer-to-be-falsy': [0],
      '@vitest/prefer-to-be-object': [0],
      '@vitest/prefer-to-be-truthy': [0],
      '@vitest/prefer-to-contain': [0],
      '@vitest/prefer-to-have-length': [0],
      '@vitest/prefer-todo': [0],
      '@vitest/require-hook': [0],
      '@vitest/require-to-throw-message': [0],
      '@vitest/require-top-level-describe': [0],
      '@vitest/valid-describe-callback': [2],
      '@vitest/valid-expect': [2],
      '@vitest/valid-title': [2],
    },
  }, {
    files: ['web_src/js/modules/fetch.js', 'web_src/js/standalone/**/*'],

    rules: {
      'no-restricted-syntax': [
        2,
        'WithStatement',
        'ForInStatement',
        'LabeledStatement',
        'SequenceExpression',
      ],
    },
  }, {
    files: ['tests/e2e/**/*.ts'],
    languageOptions: {
      globals: {
        ...globals.browser,
      },

      ecmaVersion: 'latest',
      sourceType: 'module',
    },
    rules: {
      ...playwright.configs['flat/recommended'].rules,
      'playwright/no-conditional-in-test': [0],
      'playwright/no-conditional-expect': [0],
      // allow grouping helper functions with tests
      'unicorn/consistent-function-scoping': [0],

      'playwright/no-skipped-test': [
        2,
        {
          allowConditional: true,
        },
      ],
      'playwright/no-useless-await': [2],

      'playwright/prefer-comparison-matcher': [2],
      'playwright/prefer-equality-matcher': [2],
      'playwright/prefer-native-locators': [2],
      'playwright/prefer-to-contain': [2],
      'playwright/prefer-to-have-length': [2],
      'playwright/require-to-throw-message': [2],
    },
  },
  ...vue.configs['flat/recommended'],
  {
    files: ['web_src/js/components/*.vue'],
    languageOptions: {
      globals: {
        ...globals.browser,
      },

      ecmaVersion: 'latest',
      sourceType: 'module',
    },
    rules: {
      'vue/attributes-order': [0],
      'vue/html-closing-bracket-spacing': [2, {
        startTag: 'never',
        endTag: 'never',
        selfClosingTag: 'never',
      }],
      'vue/max-attributes-per-line': [0],
      'vue-scoped-css/enforce-style-type': [0],
    },
  },

);
