import * as test from './test.js';

test.def('get prev success', async (t: test.T) => {
	const create = await t.client.createTestType({text: 'foo'});

	const get1 = await t.client.getTestType(create.id);
	t.equal(get1.text, 'foo');

	// This is test-only
	// Don't mutate objects and pass them back in GetOpts.prev
	get1.num = 5;

	const get2 = await t.client.getTestType(create.id, {prev: get1});
	t.equal(get2.text, 'foo');
	t.equal(get2.num, 5);
});
