window.addEventListener('load', async () => {
  const [{default: SwaggerUI}] = await Promise.all([
    import(/* webpackChunkName: "swagger-ui" */'swagger-ui-dist/swagger-ui-es-bundle.js'),
    import(/* webpackChunkName: "swagger-ui" */'swagger-ui-dist/swagger-ui.css'),
  ]);
  const url = document.getElementById('swagger-ui').getAttribute('data-source');

  const ui = SwaggerUI({
    url,
    dom_id: '#swagger-ui',
    deepLinking: true,
    docExpansion: 'none',
    defaultModelRendering: 'model', // don't show examples by default, because they may be incomplete
    presets: [
      SwaggerUI.presets.apis,
    ],
    plugins: [
      SwaggerUI.plugins.DownloadUrl,
    ],
  });

  window.ui = ui;
});
