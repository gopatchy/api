import * as test from './test.js';

test.def('replace success', async (t: test.T) => {
	const create = await t.client.createTestType({text: 'foo', num: 5});
	t.equal(create.text, 'foo');

	const get1 = await t.client.getTestType(create.id);
	t.equal(get1.text, 'foo');
	t.equal(get1.num, 5);

	const replace = await t.client.replaceTestType(create.id, {text: 'bar'});
	t.equal(replace.text, 'bar');

	const get2 = await t.client.getTestType(create.id);
	t.equal(get2.text, 'bar');
	t.equal(get2.num, 0);
});
