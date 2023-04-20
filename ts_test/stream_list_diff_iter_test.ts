import * as test from './test.js';

test.def('stream list diff iter success', async (t: test.T) => {
	const create = await t.client.createTestType({text: 'foo'});

	const stream = await t.client.streamListTestType({stream: 'diff'});

	try {
		await t.client.updateTestType(create.id, {text: 'bar'});

		const objs = [];

		for await (const list of stream) {
			t.equal(list.length, 1);

			objs.push(list[0]);

			if (objs.length == 2) {
				await stream.abort();
			}
		}

		t.equal(objs.map(x => x!.text), ['foo', 'bar']);
	} finally {
		await stream.close();
	}
});
