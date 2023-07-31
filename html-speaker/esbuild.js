const esbuild = require('esbuild');

esbuild.build({
    entryPoints: ['src/scripts/index.ts'],
    outdir: 'dist/scripts',
    bundle: true,
    sourcemap: true,
    minify: true,
    splitting: true,
    format: 'esm',
    target: ['esnext'],
}).catch((e) => process.exit(1));