import * as test from './test.js';

test.def('stream list diff prev success', async (t: test.T) => {
	await t.client.createTestType({text: 'foo'});
	await t.client.createTestType({text: 'bar'});

	const list = await t.client.listTestType();
	t.equal(list!.length, 2);

	// This is test-only
	// Don't mutate objects and pass them back in GetOpts.prev
	list[0]!.num = 5;

	const stream = await t.client.streamListTestType({stream: 'diff', prev: list});

	try {
		const s1 = await stream.read();
		t.true(s1);
		t.equal(s1.length, 2);
		t.equal(s1[0]!.num, 5);
	} finally {
		await stream.close();
	}
});
