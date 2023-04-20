import * as test from './test.js';

test.def('debug fetch success', async (t: test.T) => {
	const dbg = await t.client.debugInfo();
	t.true(dbg);
});
