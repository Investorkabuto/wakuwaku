import { createClient, type Interceptor } from '@connectrpc/connect';
import { createConnectTransport } from '@connectrpc/connect-web';
import { CareService } from '$lib/gen/wakuwaku/v1/care_pb';
import { authToken } from '$lib/session.svelte';

const withAuth: Interceptor = (next) => (req) => {
	const token = authToken();
	if (token) req.header.set('Authorization', `Bearer ${token}`);
	return next(req);
};

const transport = createConnectTransport({
	baseUrl: import.meta.env.VITE_API_URL ?? 'http://localhost:8080',
	interceptors: [withAuth]
});

export const care = createClient(CareService, transport);
