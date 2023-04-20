declare global {
	namespace globalThis {
		var baseURL: string;
	}
}

export function getBaseURL(): string {
	return (globalThis.baseURL as string);
}
