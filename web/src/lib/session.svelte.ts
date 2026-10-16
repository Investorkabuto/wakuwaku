// ログイン状態。現在はローカル開発用の "dev:<名前>" トークンのみ。
// Firebase Auth は Firebase プロジェクト作成後に差し替える。
const STORAGE_KEY = 'wakuwaku.devUser';

function load(): string | null {
	try {
		return localStorage.getItem(STORAGE_KEY);
	} catch {
		return null;
	}
}

export const session = $state<{ user: string | null }>({ user: load() });

export function signIn(name: string) {
	session.user = name;
	try {
		localStorage.setItem(STORAGE_KEY, name);
	} catch {
		// 保存できなくても、このタブの間はログイン状態を保つ。
	}
}

export function signOut() {
	session.user = null;
	try {
		localStorage.removeItem(STORAGE_KEY);
	} catch {
		// 無視
	}
}

export function authToken(): string | null {
	return session.user ? `dev:${session.user}` : null;
}
