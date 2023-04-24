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

const ETagKey = Symbol('etag');

export class Client {
	private baseURL: URL;
	private headers: Headers = new Headers();

	constructor(baseURL: string) {
		this.baseURL = new URL(baseURL, globalThis?.location?.href);
	}

	// Skipped: setDebug()

	setHeader(name: string, value: string) {
		this.headers.set(name, value)
	}

	resetAuth() {
		this.headers.delete('Authorization');
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

	async debugInfo(): Promise<Object> {
		const req = this.newReq('GET', '_debug');
		return req.fetchJSON();
	}

	async openAPI(): Promise<Object> {
		const req = this.newReq('GET', '_openapi');
		return req.fetchJSON();
	}

	async goClient(): Promise<string> {
		const req = this.newReq('GET', '_client.go');
		return req.fetchText();
	}

	async tsClient(): Promise<string> {
		const req = this.newReq('GET', '_client.ts');
		return req.fetchText();
	}

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

	//// Generic

	async createName<T>(name: string, obj: T): Promise<T & Metadata> {
		const req = this.newReq<T>('POST', encodeURIComponent(name));
		req.setBody(obj);
		return req.fetchObj();
	}

	async deleteName<T>(name: string, id: string, opts?: UpdateOpts<T> | null): Promise<void> {
		const req = this.newReq<T>('DELETE', `${encodeURIComponent(name)}/${encodeURIComponent(id)}`);
		req.applyUpdateOpts(opts);
		return req.fetchVoid();
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
		const req = this.newReq<T>('GET', `${encodeURIComponent(name)}/${encodeURIComponent(id)}`);
		req.applyGetOpts(opts);
		return req.fetchObj();
	}

	async listName<T>(name: string, opts?: ListOpts<T> | null): Promise<(T & Metadata)[]> {
		const req = this.newReq<T>('GET', `${encodeURIComponent(name)}`);
		req.applyListOpts(opts);
		return req.fetchList();
	}

	async replaceName<T>(name: string, id: string, obj: T, opts?: UpdateOpts<T> | null): Promise<T & Metadata> {
		const req = this.newReq<T>('PUT', `${encodeURIComponent(name)}/${encodeURIComponent(id)}`);
		req.applyUpdateOpts(opts);
		req.setBody(obj);
		return req.fetchObj();
	}

	async updateName<T>(name: string, id: string, obj: T, opts?: UpdateOpts<T> | null): Promise<T & Metadata> {
		const req = this.newReq<T>('PATCH', `${encodeURIComponent(name)}/${encodeURIComponent(id)}`);
		req.applyUpdateOpts(opts);
		req.setBody(obj);
		return req.fetchObj();
	}

	async streamGetName<T>(name: string, id: string, opts?: GetOpts<T> | null): Promise<GetStream<T>> {
		const req = this.newReq<T>('GET', `${encodeURIComponent(name)}/${encodeURIComponent(id)}`);
		req.applyGetOpts(opts);

		const controller = new AbortController();
		req.setSignal(controller.signal);

		const resp = await req.fetchStream();

		return new GetStream<T>(resp, controller, opts?.prev);
	}

	async streamListName<T>(name: string, opts?: ListOpts<T> | null): Promise<ListStream<T>> {
		const req = this.newReq<T>('GET', `${encodeURIComponent(name)}`);
		req.applyListOpts(opts);

		const controller = new AbortController();
		req.setSignal(controller.signal);

		const resp = await req.fetchStream();

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

	private newReq<T = void>(method: string, path: string): Req<T> {
		const url = new URL(path, this.baseURL);
		return new Req<T>(method, url, this.headers);
	}
}

class Scanner {
	private reader: ReadableStreamDefaultReader;
	private buf: string = '';

	constructor(stream: ReadableStream) {
		this.reader = stream.pipeThrough(new TextDecoderStream()).getReader();
	}

	async readLine(): Promise<string | null> {
		while (!this.buf.includes('\n')) {
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
		}

		const lineEnd = this.buf.indexOf('\n');
		const line = this.buf.substring(0, lineEnd);
		this.buf = this.buf.substring(lineEnd + 1);
		return line;
	}
}

class StreamEvent<T> {
	eventType: string = '';
	params:    Map<string, string> = new Map();
	data:      string = '';

	decodeObj(): T & Metadata {
		return JSON.parse(this.data);
	}

	decodeList(): (T & Metadata)[] {
		return JSON.parse(this.data);
	}
}

class EventStream<T> {
	private scan: Scanner;

	constructor(stream: ReadableStream) {
		this.scan = new Scanner(stream);
	}

	async readEvent(): Promise<StreamEvent<T> | null> {
		const data: string[] = [];
		const ev = new StreamEvent<T>();

		while (true) {
			const line = await this.scan.readLine();

			if (line == null) {
				return null;

			} else if (line.startsWith(':')) {
				continue;

			} else if (line.startsWith('event: ')) {
				ev.eventType = trimPrefix(line, 'event: ');

			} else if (line.startsWith('data: ')) {
				data.push(trimPrefix(line, 'data: '));

			} else if (line.includes(': ')) {
				const [k, v] = line.split(': ', 2);
				ev.params.set(k!, v!);

			} else if (line == '') {
				ev.data = data.join('\n');
				return ev;
			}
		}
	}
}

export class GetStream<T> {
	private eventStream: EventStream<T>;
	private controller: AbortController;
	private prev: (T & Metadata) | null;
	// TODO: Add LastEventReceived

	constructor(resp: Response, controller: AbortController, prev: (T & Metadata) | null | undefined) {
		this.eventStream = new EventStream<T>(resp.body!);
		this.controller = controller;
		this.prev = prev ?? null;
	}

	async abort() {
		this.controller.abort();
	}

	async read(): Promise<(T & Metadata) | null> {
		while (true) {
			const ev = await this.eventStream.readEvent();

			if (ev == null) {
				return null;

			} else if (ev.eventType == 'initial' || ev.eventType == 'update') {
				return ev.decodeObj();

			} else if (ev.eventType == 'notModified') {
				return this.prev;

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

export abstract class ListStream<T> {
	protected eventStream: EventStream<T>;
	private controller: AbortController;
	// TODO: Add LastEventReceived

	constructor(resp: Response, controller: AbortController) {
		this.eventStream = new EventStream<T>(resp.body!);
		this.controller = controller;
	}

	async abort() {
		this.controller.abort();
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
			const ev = await this.eventStream.readEvent();

			if (ev == null) {
				return null;

			} else if (ev.eventType == 'list') {
				return ev.decodeList();

			} else if (ev.eventType == 'notModified') {
				return this.prev;

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
			const ev = await this.eventStream.readEvent();

			if (ev == null) {
				return null;

			} else if (ev.eventType == 'add') {
				const obj = ev.decodeObj();
				this.objs.splice(parseInt(ev.params.get('new-position')!, 10), 0, obj);

			} else if (ev.eventType == 'update') {
				this.objs.splice(parseInt(ev.params.get('old-position')!, 10), 1);

				const obj = ev.decodeObj();
				this.objs.splice(parseInt(ev.params.get('new-position')!, 10), 0, obj);

			} else if (ev.eventType == 'remove') {
				this.objs.splice(parseInt(ev.params.get('old-position')!, 10), 1);

			} else if (ev.eventType == 'sync') {
				return this.objs;

			} else if (ev.eventType == 'notModified') {
				this.objs = this.prev!;
				return this.objs;

			} else if (ev.eventType == 'heartbeat') {
				continue;

			}
		}
	}
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

class Req<T> {
	private method:    string;
	private url:       URL;
	private params:    URLSearchParams;
	private headers:   Headers;
	private prevObj?:  (T & Metadata)
	private prevList?: (T & Metadata)[];
	private body?:     T;
	private signal?:   AbortSignal;

	constructor(method: string, url: URL, headers: Headers) {
		this.method = method;
		this.url = url;

		this.params = new URLSearchParams();
		this.headers = new Headers(headers);
	}

	applyGetOpts(opts: GetOpts<T> | null | undefined) {
		if (!opts) {
			return;
		}

		this.setPrevObj('If-None-Match', opts?.prev);
	}

	applyListOpts(opts: ListOpts<T> | null | undefined) {
		if (!opts) {
			return;
		}

		this.setPrevList('If-None-Match', opts?.prev);

		if (opts?.stream) {
			this.setQueryParam('_stream', opts.stream);
		}

		if (opts?.limit) {
			this.setQueryParam('_limit', `${opts.limit}`);
		}

		if (opts?.offset) {
			this.setQueryParam('_offset', `${opts.offset}`);
		}

		if (opts?.after) {
			this.setQueryParam('_after', `${opts.after}`);
		}

		for (const filter of opts?.filters || []) {
			this.setQueryParam(`${filter.path}[${filter.op}]`, filter.value);
		}

		for (const sort of opts?.sorts || []) {
			this.addQueryParam('_sort', sort);
		}
	}

	applyUpdateOpts(opts: UpdateOpts<T> | null | undefined) {
		if (!opts) {
			return;
		}

		this.setPrevObj('If-Match', opts?.prev);
	}

	setPrevObj(headerName: string, obj: (T & Metadata) | null | undefined) {
		if (!obj) {
			return;
		}

		this.headers.set(headerName, this.getETag(obj));
		this.prevObj = obj;
	}

	setPrevList(headerName: string, list: (T & Metadata)[] | null | undefined) {
		if (!list) {
			return;
		}

		this.headers.set(headerName, this.getETag(list));
		this.prevList = list;
	}

	setSignal(signal: AbortSignal) {
		this.signal = signal;
	}

	setBody(obj: T) {
		this.body = obj;
		this.headers.set('Content-Type', 'application/json');
	}

	setHeader(name: string, value: string) {
		this.headers.set(name, value);
	}

	setQueryParam(name: string, value: string) {
		this.params.set(name, value);
	}

	addQueryParam(name: string, value: string) {
		this.params.append(name, value);
	}

	async fetchObj(): Promise<T & Metadata> {
		this.headers.set('Accept', 'application/json');
		const resp = await this.fetch();

		if (this?.prevObj && resp.status == 304) {
			return this.prevObj;
		}

		await this.throwOnError(resp);

		const obj = await resp.json();
		this.setETag(obj, resp);
		return obj;
	}

	async fetchList(): Promise<(T & Metadata)[]> {
		this.headers.set('Accept', 'application/json');
		const resp = await this.fetch();

		if (this?.prevList && resp.status == 304) {
			return this.prevList;
		}

		await this.throwOnError(resp);

		const list = await resp.json();
		this.setETag(list, resp);
		return list;
	}

	async fetchJSON(): Promise<Object> {
		this.headers.set('Accept', 'application/json');
		const resp = await this.fetch();
		await this.throwOnError(resp);
		return resp.json();
	}

	async fetchText(): Promise<string> {
		this.headers.set('Accept', 'text/plain');
		const resp = await this.fetch();
		await this.throwOnError(resp);
		return resp.text();
	}

	async fetchStream(): Promise<Response> {
		this.headers.set('Accept', 'text/event-stream');
		const resp = await this.fetch();
		await this.throwOnError(resp);
		return resp;
	}

	async fetchVoid(): Promise<void> {
		const resp = await this.fetch();
		await this.throwOnError(resp);
	}

	async throwOnError(resp: Response) {
		if (!resp.ok) {
			throw new Error(await resp.json());
		}
	}

	async fetch(): Promise<Response> {
		this.url.search = `?${this.params}`;

		// TODO: Add timeout
		// TODO: Add retry strategy
		// TODO: Add Idempotency-Key support

		const reqOpts: RequestInit = {
			method: this.method,
			headers: this.headers,
			mode: 'cors',
			credentials: 'omit',
			referrerPolicy: 'no-referrer',
			keepalive: true,
			signal: this?.signal ?? null,
			body: this?.body ? JSON.stringify(this.body) : null,
		}

		const req = new Request(this.url, reqOpts);
		return fetch(req);
	}

	private getETag(obj: Object): string {
		const etag = Object.getOwnPropertyDescriptor(obj, ETagKey)?.value;

		if (!etag) {
			throw(new Error({
				messages: [
					`missing ETagKey in ${obj}`,
				],
			}));
		}

		return etag;
	}

	private setETag(obj: Object, resp: Response) {
		if (resp.headers.has('ETag')) {
			Object.defineProperty(obj, ETagKey, {
				value: resp.headers.get('ETag'),
			});
		}
	}
}

function trimPrefix(s: string, prefix: string): string {
	return s.substring(prefix.length);
}
