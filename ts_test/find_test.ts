import * as test from './test.js';

test.def('find success', async (t: test.T) => {
	const create = await t.client.createTestType({text: 'foo'});

	const find = await t.client.findTestType(create.id.substring(0, 4));
	t.equal(find.text, 'foo');
});
