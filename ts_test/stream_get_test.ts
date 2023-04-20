import * as test from './test.js';

test.def('stream get success', async (t: test.T) => {
	const create = await t.client.createTestType({text: 'foo'});

	const stream = await t.client.streamGetTestType(create.id);

	try {
		const s1 = await stream.read();
		t.equal(s1!.text, 'foo');

		await t.client.updateTestType(create.id, {text: 'bar'});

		const s2 = await stream.read();
		t.equal(s2!.text, 'bar');
	} finally {
		await stream.close();
	}
});
