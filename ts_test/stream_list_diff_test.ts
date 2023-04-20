import * as test from './test.js';

test.def('stream list diff success', async (t: test.T) => {
	await t.client.createTestType({text: 'foo'});
	await t.client.createTestType({text: 'zig'});
	await t.client.createTestType({text: 'aaa'});

	const stream = await t.client.streamListTestType({stream: 'diff', sorts: ['+text']});

	try {
		const s1 = await stream.read();
		t.true(s1);
		t.equal(s1.map(x => x.text), ['aaa', 'foo', 'zig']);

		const create2 = await t.client.createTestType({text: 'bar'});

		const s2 = await stream.read();
		t.true(s2);
		t.equal(s2.map(x => x.text), ['aaa', 'bar', 'foo', 'zig']);

		await t.client.updateTestType(create2.id, {text: 'zag'});

		const s3 = await stream.read()!;
		t.true(s3);
		t.equal(s3.map(x => x.text), ['aaa', 'foo', 'zag', 'zig']);
	} finally {
		await stream.close();
	}
});
