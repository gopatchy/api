/// <reference path="process.d.ts" />

export function getBaseURL(): string {
	return process.env['BASE_URL']!;
}
