import {optimize} from 'svgo';
import {readFile, writeFile} from 'node:fs/promises';
import {exit} from 'node:process';
import SharpConstructor from 'sharp';
import {fileURLToPath} from 'node:url';

function doExit(err) {
  if (err) console.error(err);
  exit(err ? 1 : 0);
}

async function generate(svg, path, {size, bg}) {
  const outputFile = new URL(path, import.meta.url);

  if (String(outputFile).endsWith('.svg')) {
    const {data} = optimize(svg, {
      plugins: [
        'preset-default',
        'removeDimensions',
        {
          name: 'addAttributesToSVGElement',
          params: {attributes: [{width: size}, {height: size}]},
        },
      ],
    });
    await writeFile(outputFile, data);
    return;
  }

  let sharp = (new SharpConstructor(Buffer.from(svg))).resize(size, size).png({compressionLevel: 9, palette: true, effort: 10, quality: 80});
  if (bg) {
    sharp = sharp.flatten({background: 'white'});
  }
  sharp.toFile(fileURLToPath(outputFile), (err) => err !== null && console.error(err) && exit(1));
}

async function main() {
  const logoSvg = await readFile(new URL('../assets/logo.svg', import.meta.url), 'utf8');
  const faviconSvg = await readFile(new URL('../assets/favicon.svg', import.meta.url), 'utf8');

  await Promise.all([
    generate(logoSvg, '../public/assets/img/logo.svg', {size: 32}),
    generate(logoSvg, '../public/assets/img/logo.png', {size: 512}),
    generate(faviconSvg, '../public/assets/img/favicon.svg', {size: 32}),
    generate(faviconSvg, '../public/assets/img/favicon.png', {size: 180}),
    generate(logoSvg, '../public/assets/img/avatar_default.png', {size: 200}),
    generate(logoSvg, '../public/assets/img/apple-touch-icon.png', {size: 180, bg: true}),
  ]);
}

try {
  doExit(await main());
} catch (err) {
  doExit(err);
}
