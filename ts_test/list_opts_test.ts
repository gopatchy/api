import * as test from './test.js';

test.def('list opts success', async (t: test.T) => {
	await t.client.createTestType({text: 'foo'});
	await t.client.createTestType({text: 'bar'});
	await t.client.createTestType({text: 'zig'});
	await t.client.createTestType({text: 'aaa'});

	const list = await t.client.listTestType({
		limit: 1,
		offset: 1,
		sorts: ['+text'],
		filters: [
			{
				path: 'text',
				op: 'gt',
				value: 'aaa',
			},
		],
	});
	t.equal(list.map(x => x.text), ['foo']);
});
