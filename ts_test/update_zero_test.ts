import * as test from './test.js';

test.def('update zero success', async (t: test.T) => {
	const create = await t.client.createTestType({text: 'foo', num: 5});

	await t.client.updateTestType(create.id, {num: 0});

	const get = await t.client.getTestType(create.id);
	t.equal(get.text, 'foo');
	t.equal(get.num, 0);
});
