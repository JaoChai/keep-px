import typescript from '@rollup/plugin-typescript'
import terser from '@rollup/plugin-terser'
import resolve from '@rollup/plugin-node-resolve'

export default [
  // UMD build (for <script> tag)
  {
    input: 'src/index.ts',
    output: {
      file: 'dist/pixlinks.min.js',
      format: 'umd',
      name: 'Pixlinks',
      sourcemap: true,
    },
    plugins: [
      resolve(),
      typescript({ tsconfig: './tsconfig.json' }),
      terser(),
    ],
  },
  // ESM build (for npm imports)
  {
    input: 'src/index.ts',
    output: {
      file: 'dist/pixlinks.esm.js',
      format: 'esm',
      sourcemap: true,
    },
    plugins: [
      resolve(),
      typescript({ tsconfig: './tsconfig.json' }),
    ],
  },
]
