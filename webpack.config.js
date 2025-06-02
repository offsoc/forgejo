import fastGlob from 'fast-glob';
import wrapAnsi from 'wrap-ansi';
import {init as licenseChecker} from 'license-checker-rseidelsohn';
import MiniCssExtractPlugin from 'mini-css-extract-plugin';
import MonacoWebpackPlugin from 'monaco-editor-webpack-plugin';
import {VueLoaderPlugin} from 'vue-loader';
import EsBuildLoader from 'esbuild-loader';
import {parse} from 'node:path';
import webpack from 'webpack';
import {fileURLToPath} from 'node:url';
import {readFileSync, writeFileSync} from 'node:fs';
import {env} from 'node:process';
import tailwindcss from 'tailwindcss';
import tailwindConfig from './tailwind.config.js';
import tailwindcssNesting from 'tailwindcss/nesting/index.js';
import postcssNesting from 'postcss-nesting';

const {EsbuildPlugin} = EsBuildLoader;
const {SourceMapDevToolPlugin, DefinePlugin, ProgressPlugin} = webpack;
const formatLicenseText = (licenseText) => wrapAnsi(licenseText || '', 80).trim();

const baseDirectory = import.meta.dirname;
const glob = (pattern) => fastGlob.sync(pattern, {
  cwd: baseDirectory,
  absolute: true,
});

const themes = {};
for (const path of glob('web_src/css/themes/*.css')) {
  themes[parse(path).name] = [path];
}

const isProduction = env.NODE_ENV !== 'development';

if (isProduction) {
  licenseChecker({
    start: baseDirectory,
    production: true,
    onlyAllow: 'Apache-2.0; 0BSD; BSD-2-Clause; BSD-3-Clause; BlueOak-1.0.0; MIT; ISC; Unlicense; CC-BY-4.0',
    // argparse@2.0.1 - Python-2.0. It's used in the CLI file of markdown-it and js-yaml and not in the library code.
    // idiomorph@0.3.0. See https://github.com/bigskysoftware/idiomorph/pull/37
    excludePackages: 'argparse@2.0.1;idiomorph@0.3.0',
  }, (err, dependencies) => {
    if (err) {
      throw err;
    }

    const line = '-'.repeat(80);
    const goJson = readFileSync('assets/go-licenses.json', 'utf8');
    const goModules = JSON.parse(goJson).map(({name, licenseText}) => {
      return {name, body: formatLicenseText(licenseText)};
    });
    const jsModules = Object.keys(dependencies).map((packageName) => {
      const {licenses, licenseFile} = dependencies[packageName];
      const licenseText = (licenseFile && !licenseFile.toLowerCase().includes('readme')) ? readFileSync(licenseFile) : '[no license file]';
      return {name: packageName, licenseName: licenses, body: formatLicenseText(licenseText)};
    });
    const modules = [...goModules, ...jsModules];
    const licenseTxt = modules.map(({name, licenseName, body}) => {
      const title = licenseName ? `${name} - ${licenseName}` : name;
      return `${line}\n${title}\n${line}\n${body}`;
    }).join('\n');
    writeFileSync('public/assets/licenses.txt', licenseTxt);
  });
} else {
  writeFileSync('public/assets/licenses.txt', 'Licenses are disabled during development');
}

// ENABLE_SOURCEMAP accepts the following values:
// true - all enabled, the default in development
// reduced - minimal sourcemaps, the default in production
// false - all disabled
let sourceMaps;
if ('ENABLE_SOURCEMAP' in env) {
  sourceMaps = ['true', 'false'].includes(env.ENABLE_SOURCEMAP) ? env.ENABLE_SOURCEMAP : 'reduced';
} else {
  sourceMaps = isProduction ? 'reduced' : 'true';
}

// define which web components we use for Vue to not interpret them as Vue components
const webComponents = new Set([
  // our own, in web_src/js/webcomponents
  'i18n',
  'overflow-menu',
  'origin-url',
  'absolute-date',
  'relative-time',
  // from dependencies
  'markdown-toolbar',
  'text-expander',
]);

const filterCssImport = (url, ...args) => {
  const cssFile = args[1] || args[0]; // resourcePath is 2nd argument for url and 3rd for import
  const importedFile = url.replace(/[?#].+/, '').toLowerCase();

  if (cssFile.includes('fomantic')) {
    if (/brand-icons/.test(importedFile)) return false;
    if (/(eot|ttf|otf|woff|svg)$/i.test(importedFile)) return false;
  }

  if (cssFile.includes('katex') && /(ttf|woff)$/i.test(importedFile)) {
    return false;
  }

  return true;
};

/** @type {import("webpack").Configuration} */
export default {
  mode: isProduction ? 'production' : 'development',
  entry: {
    index: [
      fileURLToPath(new URL('web_src/js/jquery.js', import.meta.url)),
      fileURLToPath(new URL('web_src/fomantic/build/semantic.js', import.meta.url)),
      fileURLToPath(new URL('web_src/js/index.js', import.meta.url)),
      fileURLToPath(new URL('node_modules/easymde/dist/easymde.min.css', import.meta.url)),
      fileURLToPath(new URL('web_src/fomantic/build/semantic.css', import.meta.url)),
      fileURLToPath(new URL('web_src/css/index.css', import.meta.url)),
    ],
    webcomponents: [
      fileURLToPath(new URL('web_src/js/webcomponents/index.js', import.meta.url)),
    ],
    forgejoswagger: [ // Forgejo swagger is OpenAPI 3.0.0 and has specific parameters
      fileURLToPath(new URL('web_src/js/standalone/forgejo-swagger.js', import.meta.url)),
      fileURLToPath(new URL('web_src/css/standalone/swagger.css', import.meta.url)),
    ],
    swagger: [
      fileURLToPath(new URL('web_src/js/standalone/swagger.js', import.meta.url)),
      fileURLToPath(new URL('web_src/css/standalone/swagger.css', import.meta.url)),
    ],
    'eventsource.sharedworker': [
      fileURLToPath(new URL('web_src/js/features/eventsource.sharedworker.js', import.meta.url)),
    ],
    ...(!isProduction && {
      devtest: [
        fileURLToPath(new URL('web_src/js/standalone/devtest.js', import.meta.url)),
        fileURLToPath(new URL('web_src/css/standalone/devtest.css', import.meta.url)),
      ],
    }),
    ...themes,
  },
  devtool: false,
  output: {
    path: fileURLToPath(new URL('public/assets', import.meta.url)),
    filename: () => 'js/[name].js',
    chunkFilename: ({chunk}) => {
      const language = (/monaco.*languages?_.+?_(.+?)_/.exec(chunk.id) || [])[1];
      return `js/${language ? `monaco-language-${language.toLowerCase()}` : `[name]`}.[contenthash:8].js`;
    },
  },
  optimization: {
    minimize: isProduction,
    minimizer: [
      new EsbuildPlugin({
        target: 'es2020',
        minify: true,
        css: true,
        legalComments: 'none',
      }),
    ],
    splitChunks: {
      chunks: 'async',
      name: (_, chunks) => chunks.map((item) => item.name).join('-'),
    },
    moduleIds: 'named',
    chunkIds: 'named',
  },
  module: {
    rules: [
      {
        test: /\.vue$/i,
        exclude: /node_modules/,
        loader: 'vue-loader',
        options: {
          compilerOptions: {
            isCustomElement: (tag) => webComponents.has(tag),
          },
        },
      },
      {
        test: /\.(js|ts)$/i,
        exclude: /node_modules/,
        use: [
          {
            loader: 'esbuild-loader',
            options: {
              loader: 'ts',
              target: 'es2020',
            },
          },
        ],
      },
      {
        test: /\.css$/i,
        use: [
          {
            loader: MiniCssExtractPlugin.loader,
          },
          {
            loader: 'css-loader',
            options: {
              sourceMap: sourceMaps === 'true',
              url: {filter: filterCssImport},
              import: {filter: filterCssImport},
              importLoaders: 1,
            },
          },
          {
            loader: 'postcss-loader',
            options: {
              postcssOptions: {
                plugins: [
                  tailwindcssNesting(postcssNesting({edition: '2024-02'})),
                  tailwindcss(tailwindConfig),
                ],
              },
            },
          },
        ],
      },
      {
        test: /\.svg$/i,
        include: fileURLToPath(new URL('public/assets/img/svg', import.meta.url)),
        type: 'asset/source',
      },
      {
        test: /\.(ttf|woff2?)$/i,
        type: 'asset/resource',
        generator: {
          filename: 'fonts/[name].[contenthash:8][ext]',
        },
      },
    ],
  },
  plugins: [
    new ProgressPlugin({
      activeModules: true,
    }),
    new webpack.ProvidePlugin({ // for htmx extensions
      htmx: 'htmx.org',
    }),
    new DefinePlugin({
      __VUE_OPTIONS_API__: true, // at the moment, many Vue components still use the Vue Options API
      __VUE_PROD_DEVTOOLS__: false, // do not enable devtools support in production
      __VUE_PROD_HYDRATION_MISMATCH_DETAILS__: false, // https://github.com/vuejs/vue-cli/pull/7443
    }),
    new VueLoaderPlugin(),
    new MiniCssExtractPlugin({
      filename: 'css/[name].css',
      chunkFilename: 'css/[name].[contenthash:8].css',
    }),
    sourceMaps !== 'false' && new SourceMapDevToolPlugin({
      filename: '[file].[contenthash:8].map',
      ...(sourceMaps === 'reduced' && {include: /^js\/index\.js$/}),
    }),
    new MonacoWebpackPlugin({
      filename: 'js/monaco-[name].[contenthash:8].worker.js',
    }),
  ],
  performance: {
    hints: false,
    maxEntrypointSize: Infinity,
    maxAssetSize: Infinity,
  },
  resolve: {
    symlinks: false,
  },
  watchOptions: {
    ignored: [
      'node_modules/**',
    ],
  },
  stats: {
    assetsSort: 'name',
    assetsSpace: Infinity,
    cached: false,
    cachedModules: false,
    children: false,
    chunkModules: false,
    chunkOrigins: false,
    chunksSort: 'name',
    colors: true,
    entrypoints: false,
    excludeAssets: [
      /^js\/monaco-language-.+\.js$/,
      !isProduction && /^licenses.txt$/,
    ].filter(Boolean),
    groupAssetsByChunk: false,
    groupAssetsByEmitStatus: false,
    groupAssetsByInfo: false,
    groupModulesByAttributes: false,
    modules: false,
    reasons: false,
    runtimeModules: false,
  },
};
