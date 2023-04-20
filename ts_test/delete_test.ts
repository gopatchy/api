import * as test from './test.js';

test.def('delete success', async (t: test.T) => {
	const create = await t.client.createTestType({text: 'foo'});
	t.equal(create.text, 'foo');

	const get = await t.client.getTestType(create.id);
	t.equal(get.text, 'foo');

	await t.client.deleteTestType(create.id);

	t.rejects(t.client.getTestType(create.id));
});
