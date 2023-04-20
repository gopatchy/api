import * as test from './test.js';

test.def('stream get iter success', async (t: test.T) => {
	const create = await t.client.createTestType({text: 'foo'});

	const stream = await t.client.streamGetTestType(create.id);

	try {
		await t.client.updateTestType(create.id, {text: 'bar'});

		const objs = [];

		for await (const obj of stream) {
			objs.push(obj);

			if (objs.length == 2) {
				await stream.abort();
			}
		}

		t.equal(objs.length, 2);
		t.equal(objs[0]!.text, 'foo');
		t.equal(objs[1]!.text, 'bar');
	} finally {
		await stream.close();
	}
});
