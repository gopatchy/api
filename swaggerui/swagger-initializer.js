addEventListener('load', async () => {
	const oa = await fetch('../_openapi');
	const oajs = await oa.json();

	SwaggerUIBundle({
		url: oajs.servers[0].url + '/_openapi',
		dom_id: '#swagger-ui',
		layout: "BaseLayout",
	});
});
