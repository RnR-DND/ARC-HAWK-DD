import nextCoreWebVitals from 'eslint-config-next/core-web-vitals';

const config = [
  {
    ignores: [
      '.next/**',
      'node_modules/**',
      'out/**',
      'public/**',
      '**/*.d.ts',
      'coverage/**',
      'playwright-report/**',
      'test-results/**',
    ],
  },
  ...nextCoreWebVitals,
];

export default config;
