{{- if and .Info .Info.Title -}}
// {{ .Info.Title }} client
{{- end }}

{{- range $type := .Types }}

export interface {{ $type.TypeUpperCamel }} {
	{{- range $field := .Fields }}
	{{ padRight (printf "%s?:" $field.NameLowerCamel) (add $type.FieldNameMaxLen 2) }} {{ $field.TSType }};
	{{- end }}
}

{{- end }}

export interface Metadata {
	id:           string;
	etag:         string;
	generation:   number;
}

export interface GetOpts<T> {
	prev?: T & Metadata;
}

export interface ListOpts<T> {
	stream?:  string;
	limit?:   number;
	offset?:  number;
	after?:   string;
	sorts?:   string[];
	filters?: Filter[];
	prev?:    (T & Metadata)[];
}

export interface Filter {
	path:   string;
	op:     string;
	value:  string;
}

export interface UpdateOpts<T> {
	prev?: T & Metadata;
}

export interface JSONError {
	messages:  string[];
}

interface FetchOptions {
	params?:  URLSearchParams;
	headers?: Headers;
	prev?:    any;
	body?:    any;
	signal?:  AbortSignal;
	stream?:  boolean;
}

interface StreamEvent {
	eventType: string;
	params:    Map<string, string>;
	data:      string;
}

const ETagKey = Symbol('etag');

class StreamCore {
	private reader: ReadableStreamDefaultReader;
	private controller: AbortController;

	private buf: string = '';

	constructor(resp: Response, controller: AbortController) {
		this.reader = resp.body!.pipeThrough(new TextDecoderStream()).getReader();
		this.controller = controller;
	}

	async abort() {
		this.controller.abort();
	}

	protected async readEvent(): Promise<StreamEvent | null> {
		const data: string[] = [];
		const ev: StreamEvent = {
			eventType: '',
			params: new Map(),
			data: '',
		};

		while (true) {
			const line = await this.readLine();

			if (line == null) {
				return null;

			} else if (line.startsWith(':')) {
				continue;

			} else if (line.startsWith('event: ')) {
				ev.eventType = this.removePrefix(line, 'event: ');

			} else if (line.startsWith('data: ')) {
				data.push(this.removePrefix(line, 'data: '));

			} else if (line.includes(': ')) {
				const [k, v] = line.split(': ', 2);
				ev.params.set(k!, v!);

			} else if (line == '') {
				ev.data = data.join('\n');
				return ev;
			}
		}
	}

	private async readLine(): Promise<string | null> {
		while (true) {
			const lineEnd = this.buf.indexOf('\n');

			if (lineEnd == -1) {
				let chunk: ReadableStreamReadResult<any>;

				try {
					chunk = await this.reader.read();
				} catch {
					return null;
				}

				if (chunk.done) {
					return null;
				}

				this.buf += chunk.value;
				continue;
			}

			const line = this.buf.substring(0, lineEnd);
			this.buf = this.buf.substring(lineEnd + 1);

			return line;
		}
	}

	private removePrefix(s: string, prefix: string): string {
		return s.substring(prefix.length);
	}
}

export class GetStream<T> extends StreamCore {
	private prev: (T & Metadata) | null;

	constructor(resp: Response, controller: AbortController, prev: (T & Metadata) | null | undefined) {
		super(resp, controller);

		this.prev = prev ?? null;
	}

	async read(): Promise<(T & Metadata) | null> {
		while (true) {
			const ev = await this.readEvent();

			if (ev == null) {
				return null;

			} else if (ev.eventType == 'initial' || ev.eventType == 'update') {
				return JSON.parse(ev.data);

			} else if (ev.eventType == 'notModified') {
				if (this.prev == null) {
					throw new Error({
						messages: [
							'notModified without previous',
						],
					});
				}

				const prev = this.prev;
				this.prev = null;

				return prev;

			} else if (ev.eventType == 'heartbeat') {
				continue;

			}
		}
	}

	async close() {
		this.abort();

		for await (const _ of this) {}
	}

	async *[Symbol.asyncIterator](): AsyncIterableIterator<T & Metadata> {
		while (true) {
			const obj = await this.read();

			if (obj == null) {
				return;
			}

			yield obj;
		}
	}
}

export abstract class ListStream<T> extends StreamCore {
	constructor(resp: Response, controller: AbortController) {
		super(resp, controller);
	}

	async close() {
		this.abort();

		for await (const _ of this) {}
	}

	abstract read(): Promise<(T & Metadata)[] | null>;

	async *[Symbol.asyncIterator](): AsyncIterableIterator<(T & Metadata)[]> {
		while (true) {
			const list = await this.read();

			if (list == null) {
				return;
			}

			yield list;
		}
	}
}

export class ListStreamFull<T> extends ListStream<T> {
	private prev: (T & Metadata)[] | null;

	constructor(resp: Response, controller: AbortController, prev: (T & Metadata)[] | null | undefined) {
		super(resp, controller);
		this.prev = prev ?? null;
	}

	async read(): Promise<(T & Metadata)[] | null> {
		while (true) {
			const ev = await this.readEvent();

			if (ev == null) {
				return null;

			} else if (ev.eventType == 'list') {
				return JSON.parse(ev.data);

			} else if (ev.eventType == 'notModified') {
				if (this.prev == null) {
					throw new Error({
						messages: [
							'notModified without previous',
						],
					});
				}

				const prev = this.prev;
				this.prev = null;

				return prev;

			} else if (ev.eventType == 'heartbeat') {
				continue;

			}
		}
	}
}

export class ListStreamDiff<T> extends ListStream<T> {
	private prev: (T & Metadata)[] | null;
	private objs: (T & Metadata)[] = [];

	constructor(resp: Response, controller: AbortController, prev: (T & Metadata)[] | null | undefined) {
		super(resp, controller);
		this.prev = prev ?? null;
	}

	async read(): Promise<(T & Metadata)[] | null> {
		while (true) {
			const ev = await this.readEvent();

			if (ev == null) {
				return null;

			} else if (ev.eventType == 'add') {
				const obj = JSON.parse(ev.data) as T & Metadata;
				this.objs.splice(parseInt(ev.params.get('new-position')!, 10), 0, obj);

			} else if (ev.eventType == 'update') {
				this.objs.splice(parseInt(ev.params.get('old-position')!, 10), 1);

				const obj = JSON.parse(ev.data) as T & Metadata;
				this.objs.splice(parseInt(ev.params.get('new-position')!, 10), 0, obj);

			} else if (ev.eventType == 'remove') {
				this.objs.splice(parseInt(ev.params.get('old-position')!, 10), 1);

			} else if (ev.eventType == 'sync') {
				return this.objs;

			} else if (ev.eventType == 'notModified') {
				if (!this.prev) {
					throw new Error({
						messages: [
							"notModified without prev",
						],
					});
				}

				this.objs = this.prev;
				return this.objs;

			} else if (ev.eventType == 'heartbeat') {
				continue;

			}
		}
	}
}

class ClientCore {
	protected baseURL: URL;
	protected headers: Headers = new Headers();

	constructor(baseURL: string) {
		this.baseURL = new URL(baseURL, globalThis?.location?.href);
	}

	async debugInfo(): Promise<Object> {
		return this.fetch('GET', '_debug');
	}

	//// Generic

	async createName<T>(name: string, obj: T): Promise<T & Metadata> {
		return this.fetch(
			'POST',
			encodeURIComponent(name),
			{
				body: obj,
			},
		);
	}

	async deleteName<T>(name: string, id: string, opts?: UpdateOpts<T> | null): Promise<void> {
		return this.fetch(
			'DELETE',
			`${encodeURIComponent(name)}/${encodeURIComponent(id)}`,
			{
				headers: this.buildUpdateHeaders(opts),
			},
		);
	}

	async findName<T>(name: string, shortID: string): Promise<T & Metadata> {
		const opts: ListOpts<T> = {
			filters: [
				{
					path: 'id',
					op: 'hp',
					value: shortID,
				},
			],
		};

		const list = await this.listName<T>(name, opts);

		if (list.length != 1) {
			throw new Error({
				messages: [
					'not found',
				],
			});
		}

		return list[0]!;
	}

	async getName<T>(name: string, id: string, opts?: GetOpts<T> | null): Promise<T & Metadata> {
		return this.fetch(
			'GET',
			`${encodeURIComponent(name)}/${encodeURIComponent(id)}`,
			{
				headers: this.buildGetHeaders(opts),
				prev: opts?.prev,
			},
		);
	}

	async listName<T>(name: string, opts?: ListOpts<T> | null): Promise<(T & Metadata)[]> {
		return this.fetch(
			'GET',
			`${encodeURIComponent(name)}`,
			{
				params: this.buildListParams(opts),
				headers: this.buildListHeaders(opts),
				prev: opts?.prev,
			},
		);
	}

	async replaceName<T>(name: string, id: string, obj: T, opts?: UpdateOpts<T> | null): Promise<T & Metadata> {
		return this.fetch(
			'PUT',
			`${encodeURIComponent(name)}/${encodeURIComponent(id)}`,
			{
				headers: this.buildUpdateHeaders(opts),
				body: obj,
			},
		);
	}

	async updateName<T>(name: string, id: string, obj: T, opts?: UpdateOpts<T> | null): Promise<T & Metadata> {
		return this.fetch(
			'PATCH',
			`${encodeURIComponent(name)}/${encodeURIComponent(id)}`,
			{
				headers: this.buildUpdateHeaders(opts),
				body: obj,
			},
		);
	}

	async streamGetName<T>(name: string, id: string, opts?: GetOpts<T> | null): Promise<GetStream<T>> {
		const controller = new AbortController();

		const resp = await this.fetch(
			'GET',
			`${encodeURIComponent(name)}/${encodeURIComponent(id)}`,
			{
				headers: this.buildGetHeaders(opts),
				stream: true,
				signal: controller.signal,
			},
		);

		return new GetStream<T>(resp, controller, opts?.prev);
	}

	async streamListName<T>(name: string, opts?: ListOpts<T> | null): Promise<ListStream<T>> {
		const controller = new AbortController();

		const resp = await this.fetch(
			'GET',
			`${encodeURIComponent(name)}`,
			{
				params: this.buildListParams(opts),
				headers: this.buildListHeaders(opts),
				stream: true,
				signal: controller.signal,
			},
		);

		try {
			switch (resp.headers.get('Stream-Format')) {
			case 'full':
				return new ListStreamFull<T>(resp, controller, opts?.prev);

			case 'diff':
				return new ListStreamDiff<T>(resp, controller, opts?.prev);

			default:
				throw new Error({
					messages: [
						`invalid Stream-Format: ${resp.headers.get('Stream-Format')}`,
					],
				});
			}
		} catch (e) {
			controller.abort();
			throw e;
		}
	}

	private buildListParams<T>(opts: ListOpts<T> | null | undefined): URLSearchParams {
		const params = new URLSearchParams();

		if (!opts) {
			return params;
		}

		if (opts.stream) {
			params.set('_stream', opts.stream);
		}

		if (opts.limit) {
			params.set('_limit', `${opts.limit}`);
		}

		if (opts.offset) {
			params.set('_offset', `${opts.offset}`);
		}

		if (opts.after) {
			params.set('_after', `${opts.after}`);
		}

		for (const filter of opts.filters || []) {
			params.set(`${filter.path}[${filter.op}]`, filter.value);
		}

		for (const sort of opts.sorts || []) {
			params.append('_sort', sort);
		}

		return params;
	}

	private buildListHeaders<T>(opts: ListOpts<T> | null | undefined): Headers {
		const headers = new Headers();

		this.addETagHeader(headers, 'If-None-Match', opts?.prev);

		return headers;
	}

	private buildGetHeaders<T>(opts: GetOpts<T> | null | undefined): Headers {
		const headers = new Headers();

		this.addETagHeader(headers, 'If-None-Match', opts?.prev);

		return headers;
	}

	private buildUpdateHeaders<T>(opts: UpdateOpts<T> | null | undefined): Headers {
		const headers = new Headers();

		this.addETagHeader(headers, 'If-Match', opts?.prev);

		return headers;
	}

	private addETagHeader(headers: Headers, name: string, obj: any | undefined) {
		if (!obj) {
			return;
		}

		const etag = Object.getOwnPropertyDescriptor(obj, ETagKey)?.value;

		if (!etag) {
			throw(new Error({
				messages: [
					`missing ETagKey in ${obj}`,
				],
			}));
		}

		headers.set(name, etag);
	}

	protected async fetch(method: string, path: string, opts?: FetchOptions): Promise<any> {
		const url = new URL(path, this.baseURL);

		if (opts?.params) {
			url.search = `?${opts.params}`;
		}

		// TODO: Add timeout
		// TODO: Add retry strategy
		// TODO: Add Idempotency-Key support

		const reqOpts: RequestInit = {
			method: method,
			headers: new Headers(this.headers),
			mode: 'cors',
			credentials: 'omit',
			referrerPolicy: 'no-referrer',
			keepalive: true,
			signal: opts?.signal ?? null,
		}

		if (opts?.headers) {
			for (const [k, v] of opts.headers) {
				(<Headers>reqOpts.headers).append(k, v);
			}
		}

		if (opts?.body) {
			reqOpts.body = JSON.stringify(opts.body);
			(<Headers>reqOpts.headers).set('Content-Type', 'application/json');
		}

		if (opts?.stream) {
			(<Headers>reqOpts.headers).set('Accept', 'text/event-stream');
		}

		const req = new Request(url, reqOpts);

		const resp = await fetch(req);

		if (opts?.prev && resp.status == 304) {
			return opts.prev;
		}

		if (!resp.ok) {
			throw new Error(await resp.json());
		}

		if (resp.status == 200) {
			if (opts?.stream) {
				return resp;
			}

			const js = await resp.json();

			if (resp.headers.has('ETag')) {
				Object.defineProperty(js, ETagKey, {
					value: resp.headers.get('ETag'),
				});
			}

			return js;
		}
	}
}

export class Client extends ClientCore {
	constructor(baseURL: string) {
		super(baseURL);
	}

	{{- if .AuthBasic }}

	setBasicAuth(user: string, pass: string) {
		const enc = btoa(`${user}:${pass}`);
		this.headers.set('Authorization', `Basic ${enc}`);
	}
	{{- end }}

	{{- if .AuthBearer }}

	setAuthToken(token: string) {
		this.headers.set('Authorization', `Bearer ${token}`);
	}
	{{- end }}

	{{- range $api := .APIs }}

	//// {{ $api.NameUpperCamel }}

	async create{{ $api.NameUpperCamel }}(obj: {{ $api.TypeUpperCamel }}): Promise<{{ $api.TypeUpperCamel }} & Metadata> {
		return this.createName<{{ $api.TypeUpperCamel }}>('{{ $api.NameLower }}', obj);
	}

	async delete{{ $api.NameUpperCamel }}(id: string, opts?: UpdateOpts<{{ $api.TypeUpperCamel }}> | null): Promise<void> {
		return this.deleteName('{{ $api.NameLower }}', id, opts);
	}

	async find{{ $api.NameUpperCamel }}(shortID: string): Promise<{{ $api.TypeUpperCamel }} & Metadata> {
		return this.findName<{{ $api.TypeUpperCamel }}>('{{ $api.NameLower }}', shortID);
	}

	async get{{ $api.NameUpperCamel }}(id: string, opts?: GetOpts<{{ $api.TypeUpperCamel }}> | null): Promise<{{ $api.TypeUpperCamel }} & Metadata> {
		return this.getName<{{ $api.TypeUpperCamel }}>('{{ $api.NameLower }}', id, opts);
	}

	async list{{ $api.NameUpperCamel }}(opts?: ListOpts<{{ $api.TypeUpperCamel }}> | null): Promise<({{ $api.TypeUpperCamel }} & Metadata)[]> {
		return this.listName<{{ $api.TypeUpperCamel }}>('{{ $api.NameLower }}', opts);
	}

	async replace{{ $api.NameUpperCamel }}(id: string, obj: {{ $api.TypeUpperCamel }}, opts?: UpdateOpts<{{ $api.TypeUpperCamel }}> | null): Promise<{{ $api.TypeUpperCamel }} & Metadata> {
		return this.replaceName<{{ $api.TypeUpperCamel }}>('{{ $api.NameLower }}', id, obj, opts);
	}

	async update{{ $api.NameUpperCamel }}(id: string, obj: {{ $api.TypeUpperCamel }}, opts?: UpdateOpts<{{ $api.TypeUpperCamel }}> | null): Promise<{{ $api.TypeUpperCamel }} & Metadata> {
		return this.updateName<{{ $api.TypeUpperCamel }}>('{{ $api.NameLower }}', id, obj, opts);
	}

	async streamGet{{ $api.NameUpperCamel }}(id: string, opts?: GetOpts<{{ $api.TypeUpperCamel }}> | null): Promise<GetStream<{{ $api.TypeUpperCamel }}>> {
		return this.streamGetName<{{ $api.TypeUpperCamel }}>('{{ $api.NameLower }}', id, opts);
	}

	async streamList{{ $api.NameUpperCamel }}(opts?: ListOpts<{{ $api.TypeUpperCamel }}> | null): Promise<ListStream<{{ $api.TypeUpperCamel }}>> {
		return this.streamListName<{{ $api.TypeUpperCamel }}>('{{ $api.NameLower }}', opts);
	}

	{{- end }}
}

export class Error {
	messages: string[];

	constructor(json: JSONError) {
		this.messages = json.messages;
	}

	toString(): string {
		return this.messages[0] ?? 'error';
	}
}
