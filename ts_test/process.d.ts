declare module "node:process" {
	global {
		var process: Process;
	}

	interface Process {
		env: Env;
	}

	interface Env {
		[key: string]: string;
	}
}
