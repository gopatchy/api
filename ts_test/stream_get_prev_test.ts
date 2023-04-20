import * as test from './test.js';

test.def('stream get prev success', async (t: test.T) => {
	const create = await t.client.createTestType({text: 'foo'});

	// This is test-only
	// Don't mutate objects and pass them back in GetOpts.prev
	create.num = 5;

	const stream = await t.client.streamGetTestType(create.id, {prev: create});

	try {
		const s1 = await stream.read();
		t.equal(s1!.text, 'foo');
		t.equal(s1!.num, 5);
	} finally {
		await stream.close();
	}
});
