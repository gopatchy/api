import * as test from './test.js';

test.def('stream list full success', async (t: test.T) => {
	await t.client.createTestType({text: 'foo'});

	const stream = await t.client.streamListTestType({sorts: ['+text']});

	try {
		const s1 = await stream.read();
		t.true(s1);
		t.equal(s1.length, 1);
		t.equal(s1[0]!.text, 'foo');

		const create2 = await t.client.createTestType({text: 'bar'});

		const s2 = await stream.read();
		t.true(s2);
		t.equal(s2.length, 2);
		t.equal(s2[0]!.text, 'bar');
		t.equal(s2[1]!.text, 'foo');

		await t.client.updateTestType(create2.id, {text: 'zig'});

		const s3 = await stream.read()!;
		t.true(s3);
		t.equal(s3.length, 2);
		t.equal(s3[0]!.text, 'foo');
		t.equal(s3[1]!.text, 'zig');
	} finally {
		await stream.close();
	}
});
