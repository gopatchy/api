import * as test from './test.js';

test.def('delete prev failure', async (t: test.T) => {
	const create = await t.client.createTestType({text: 'foo'});
	const get = await t.client.getTestType(create.id);
	await t.client.updateTestType(create.id, {text: 'bar'});

	t.rejects(t.client.deleteTestType(create.id, {prev: get}));

	await t.client.getTestType(create.id);
});
