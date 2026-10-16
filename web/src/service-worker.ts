/// <reference types="@sveltejs/kit" />
/// <reference no-default-lib="true"/>
/// <reference lib="esnext" />
/// <reference lib="webworker" />

import { build, files, version } from '$service-worker';

const sw = self as unknown as ServiceWorkerGlobalScope;
const CACHE = `wakuwaku-${version}`;
const ASSETS = [...build, ...files];
// SPAの本体（adapter-static の fallback）。オフライン時のページ遷移で返す。
const SHELL = '/';

sw.addEventListener('install', (event) => {
	event.waitUntil(caches.open(CACHE).then((cache) => cache.addAll([...ASSETS, SHELL])));
});

sw.addEventListener('activate', (event) => {
	event.waitUntil(
		caches
			.keys()
			.then((keys) => Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k))))
	);
});

// アプリの静的ファイルだけをキャッシュから返す。APIは常にネットワークへ。
sw.addEventListener('fetch', (event) => {
	if (event.request.method !== 'GET') return;
	const url = new URL(event.request.url);
	if (url.origin !== sw.location.origin) return;
	if (ASSETS.includes(url.pathname)) {
		event.respondWith(caches.match(url.pathname).then((r) => r ?? fetch(event.request)));
	} else if (event.request.mode === 'navigate') {
		// SPA: ページ遷移はオフライン時もアプリ本体を返す。
		event.respondWith(fetch(event.request).catch(() => caches.match(SHELL) as Promise<Response>));
	}
});

// Webプッシュ（服薬リマインド）。送信側は次の段階で実装する。
sw.addEventListener('push', (event) => {
	const data = event.data?.json() ?? {};
	event.waitUntil(
		sw.registration.showNotification(data.title ?? 'わくわく農園', {
			body: data.body ?? 'おくすりの時間です',
			icon: '/icon.svg',
			tag: data.tag,
			data: { url: data.url ?? '/' }
		})
	);
});

sw.addEventListener('notificationclick', (event) => {
	event.notification.close();
	const url = event.notification.data?.url ?? '/';
	event.waitUntil(
		sw.clients.matchAll({ type: 'window' }).then((clients) => {
			const open = clients.find((c) => new URL(c.url).origin === sw.location.origin);
			return open ? open.focus().then((c) => c.navigate(url)) : sw.clients.openWindow(url);
		})
	);
});
