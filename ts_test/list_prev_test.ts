import * as test from './test.js';

test.def('list prev success', async (t: test.T) => {
	await t.client.createTestType({text: 'foo'});
	await t.client.createTestType({text: 'bar'});

	const list1 = await t.client.listTestType();
	t.equal(list1.map(x => x.text).sort(), ['bar', 'foo']);

	// This is test-only
	// Don't mutate lists and pass them back in ListOpts.prev
	list1[0]!.num = 5;

	const list2 = await t.client.listTestType({prev: list1});
	t.equal(list2.map(x => x.text).sort(), ['bar', 'foo']);
	t.equal(list2[0]!.num, 5);
});
