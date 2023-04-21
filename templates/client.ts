{{- if and .Info .Info.Title -}}
// {{ .Info.Title }} client
{{- end }}

{{- range $type := .Types }}
{{- if $type.NameLower }}

export interface {{ $type.TypeUpperCamel }}Request {
	{{- range $field := .Fields }}
	{{ padRight (printf "%s?:" $field.NameLowerCamel) (add $type.FieldNameMaxLen 2) }} {{ $field.TSType }};
	{{- end }}
}

export interface {{ $type.TypeUpperCamel }}Response extends MetadataResponse {
	{{- range $field := .Fields }}
	{{- if $field.Optional }}
	{{ padRight (printf "%s?:" $field.NameLowerCamel) (add $type.FieldNameMaxLen 2) }} {{ $field.TSType }};
	{{- else }}
	{{ padRight (printf "%s:" $field.NameLowerCamel) (add $type.FieldNameMaxLen 2) }} {{ $field.TSType }};
	{{- end }}
	{{- end }}
}

{{- else }}

export interface {{ $type.TypeUpperCamel }} {
	{{- range $field := .Fields }}
	{{ padRight (printf "%s?:" $field.NameLowerCamel) (add $type.FieldNameMaxLen 2) }} {{ $field.TSType }};
	{{- end }}
}
{{- end }}
{{- end }}

export interface MetadataResponse {
	id:           string;
	etag:         string;
	generation:   number;
}

export interface GetOpts<T extends MetadataResponse> {
	prev?: T;
}

export interface ListOpts<T extends MetadataResponse> {
	stream?:  string;
	limit?:   number;
	offset?:  number;
	after?:   string;
	sorts?:   string[];
	filters?: Filter[];
	prev?:    T[];
}

export interface Filter {
	path:   string;
	op:     string;
	value:  string;
}

export interface UpdateOpts<T extends MetadataResponse> {
	prev?: T;
}

export interface JSONError {
	messages:  string[];
}

export interface DebugInfo {
	server: ServerInfo;
	ip:     IPInfo;
	http:   HTTPInfo;
	tls:    TLSInfo;
}

export interface ServerInfo {
	hostname:  string;
}

export interface IPInfo {
	remoteAddr:  string;
}

export interface HTTPInfo {
	protocol:  string;
	method:    string;
	header:    string;
	url:       string;
}

export interface TLSInfo {
	version:             number;
	didResume:           boolean;
	cipherSuite:         number;
	negotiatedProtocol:  string;
	serverName:          string;
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

export class GetStream<T extends MetadataResponse> extends StreamCore {
	private prev: T | null;

	constructor(resp: Response, controller: AbortController, prev: T | null | undefined) {
		super(resp, controller);

		this.prev = prev ?? null;
	}

	async read(): Promise<T | null> {
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

	async *[Symbol.asyncIterator](): AsyncIterableIterator<T> {
		while (true) {
			const obj = await this.read();

			if (obj == null) {
				return;
			}

			yield obj;
		}
	}
}

export abstract class ListStream<T extends MetadataResponse> extends StreamCore {
	constructor(resp: Response, controller: AbortController) {
		super(resp, controller);
	}

	async close() {
		this.abort();

		for await (const _ of this) {}
	}

	abstract read(): Promise<T[] | null>;

	async *[Symbol.asyncIterator](): AsyncIterableIterator<T[]> {
		while (true) {
			const list = await this.read();

			if (list == null) {
				return;
			}

			yield list;
		}
	}
}

export class ListStreamFull<T extends MetadataResponse> extends ListStream<T> {
	private prev: T[] | null;

	constructor(resp: Response, controller: AbortController, prev: T[] | null | undefined) {
		super(resp, controller);
		this.prev = prev ?? null;
	}

	async read(): Promise<T[] | null> {
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

export class ListStreamDiff<T extends MetadataResponse> extends ListStream<T> {
	private prev: T[] | null;
	private objs: T[] = [];

	constructor(resp: Response, controller: AbortController, prev: T[] | null | undefined) {
		super(resp, controller);
		this.prev = prev ?? null;
	}

	async read(): Promise<T[] | null> {
		while (true) {
			const ev = await this.readEvent();

			if (ev == null) {
				return null;

			} else if (ev.eventType == 'add') {
				const obj = JSON.parse(ev.data) as T;
				this.objs.splice(parseInt(ev.params.get('new-position')!, 10), 0, obj);

			} else if (ev.eventType == 'update') {
				this.objs.splice(parseInt(ev.params.get('old-position')!, 10), 1);

				const obj = JSON.parse(ev.data) as T;
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

	async debugInfo(): Promise<DebugInfo> {
		return this.fetch('GET', '_debug');
	}

	//// Generic

	async createName<TOut extends MetadataResponse, TIn>(name: string, obj: TIn): Promise<TOut> {
		return this.fetch(
			'POST',
			encodeURIComponent(name),
			{
				body: obj,
			},
		);
	}

	async deleteName<TOut extends MetadataResponse>(name: string, id: string, opts?: UpdateOpts<TOut> | null): Promise<void> {
		return this.fetch(
			'DELETE',
			`${encodeURIComponent(name)}/${encodeURIComponent(id)}`,
			{
				headers: this.buildUpdateHeaders(opts),
			},
		);
	}

	async findName<TOut extends MetadataResponse>(name: string, shortID: string): Promise<TOut> {
		const opts: ListOpts<TOut> = {
			filters: [
				{
					path: 'id',
					op: 'hp',
					value: shortID,
				},
			],
		};

		const list = await this.listName<TOut>(name, opts);

		if (list.length != 1) {
			throw new Error({
				messages: [
					'not found',
				],
			});
		}

		return list[0]!;
	}

	async getName<TOut extends MetadataResponse>(name: string, id: string, opts?: GetOpts<TOut> | null): Promise<TOut> {
		return this.fetch(
			'GET',
			`${encodeURIComponent(name)}/${encodeURIComponent(id)}`,
			{
				headers: this.buildGetHeaders(opts),
				prev: opts?.prev,
			},
		);
	}

	async listName<TOut extends MetadataResponse>(name: string, opts?: ListOpts<TOut> | null): Promise<TOut[]> {
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

	async replaceName<TOut extends MetadataResponse, TIn>(name: string, id: string, obj: TIn, opts?: UpdateOpts<TOut> | null): Promise<TOut> {
		return this.fetch(
			'PUT',
			`${encodeURIComponent(name)}/${encodeURIComponent(id)}`,
			{
				headers: this.buildUpdateHeaders(opts),
				body: obj,
			},
		);
	}

	async updateName<TOut extends MetadataResponse, TIn>(name: string, id: string, obj: TIn, opts?: UpdateOpts<TOut> | null): Promise<TOut> {
		return this.fetch(
			'PATCH',
			`${encodeURIComponent(name)}/${encodeURIComponent(id)}`,
			{
				headers: this.buildUpdateHeaders(opts),
				body: obj,
			},
		);
	}

	async streamGetName<TOut extends MetadataResponse>(name: string, id: string, opts?: GetOpts<TOut> | null): Promise<GetStream<TOut>> {
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

		return new GetStream<TOut>(resp, controller, opts?.prev);
	}

	async streamListName<TOut extends MetadataResponse>(name: string, opts?: ListOpts<TOut> | null): Promise<ListStream<TOut>> {
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
				return new ListStreamFull<TOut>(resp, controller, opts?.prev);

			case 'diff':
				return new ListStreamDiff<TOut>(resp, controller, opts?.prev);

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

	private buildListParams<T extends MetadataResponse>(opts: ListOpts<T> | null | undefined): URLSearchParams {
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

	private buildListHeaders<T extends MetadataResponse>(opts: ListOpts<T> | null | undefined): Headers {
		const headers = new Headers();

		this.addETagHeader(headers, 'If-None-Match', opts?.prev);

		return headers;
	}

	private buildGetHeaders<T extends MetadataResponse>(opts: GetOpts<T> | null | undefined): Headers {
		const headers = new Headers();

		this.addETagHeader(headers, 'If-None-Match', opts?.prev);

		return headers;
	}

	private buildUpdateHeaders<T extends MetadataResponse>(opts: UpdateOpts<T> | null | undefined): Headers {
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

	{{- range $type := .Types }}
	{{- if not $type.NameLower }} {{- continue }} {{- end }}

	//// {{ $type.NameUpperCamel }}

	async create{{ $type.NameUpperCamel }}(obj: {{ $type.TypeUpperCamel }}Request): Promise<{{ $type.TypeUpperCamel }}Response> {
		return this.createName<{{ $type.TypeUpperCamel }}Response, {{ $type.TypeUpperCamel }}Request>('{{ $type.NameLower }}', obj);
	}

	async delete{{ $type.NameUpperCamel }}(id: string, opts?: UpdateOpts<{{ $type.TypeUpperCamel }}Response> | null): Promise<void> {
		return this.deleteName('{{ $type.NameLower }}', id, opts);
	}

	async find{{ $type.NameUpperCamel }}(shortID: string): Promise<{{ $type.TypeUpperCamel }}Response> {
		return this.findName<{{ $type.TypeUpperCamel }}Response>('{{ $type.NameLower }}', shortID);
	}

	async get{{ $type.NameUpperCamel }}(id: string, opts?: GetOpts<{{ $type.TypeUpperCamel }}Response> | null): Promise<{{ $type.TypeUpperCamel }}Response> {
		return this.getName<{{ $type.TypeUpperCamel }}Response>('{{ $type.NameLower }}', id, opts);
	}

	async list{{ $type.NameUpperCamel }}(opts?: ListOpts<{{ $type.TypeUpperCamel }}Response> | null): Promise<{{ $type.TypeUpperCamel }}Response[]> {
		return this.listName<{{ $type.TypeUpperCamel }}Response>('{{ $type.NameLower }}', opts);
	}

	async replace{{ $type.NameUpperCamel }}(id: string, obj: {{ $type.TypeUpperCamel }}Request, opts?: UpdateOpts<{{ $type.TypeUpperCamel }}Response> | null): Promise<{{ $type.TypeUpperCamel }}Response> {
		return this.replaceName<{{ $type.TypeUpperCamel }}Response, {{ $type.TypeUpperCamel }}Request>('{{ $type.NameLower }}', id, obj, opts);
	}

	async update{{ $type.NameUpperCamel }}(id: string, obj: {{ $type.TypeUpperCamel }}Request, opts?: UpdateOpts<{{ $type.TypeUpperCamel }}Response> | null): Promise<{{ $type.TypeUpperCamel }}Response> {
		return this.updateName<{{ $type.TypeUpperCamel }}Response, {{ $type.TypeUpperCamel }}Request>('{{ $type.NameLower }}', id, obj, opts);
	}

	async streamGet{{ $type.NameUpperCamel }}(id: string, opts?: GetOpts<{{ $type.TypeUpperCamel }}Response> | null): Promise<GetStream<{{ $type.TypeUpperCamel }}Response>> {
		return this.streamGetName<{{ $type.TypeUpperCamel }}Response>('{{ $type.NameLower }}', id, opts);
	}

	async streamList{{ $type.NameUpperCamel }}(opts?: ListOpts<{{ $type.TypeUpperCamel }}Response> | null): Promise<ListStream<{{ $type.TypeUpperCamel }}Response>> {
		return this.streamListName<{{ $type.TypeUpperCamel }}Response>('{{ $type.NameLower }}', opts);
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
