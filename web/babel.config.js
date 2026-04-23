module.exports = {
  presets: [
    [
      '@babel/preset-env',
      {
        targets: {
          browsers: ['> 1%', 'last 2 versions', 'not dead'],
        },
        useBuiltIns: 'entry',
        corejs: 3,
      },
    ],
  ],
  plugins: [['@babel/plugin-transform-optional-chaining', { loose: false }]],
};
