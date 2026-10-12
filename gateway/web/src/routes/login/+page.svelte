<script lang="ts">
	import { goto } from '$app/navigation';
	import { session } from '$lib/session.svelte';
	import { ApiError } from '$lib/api';
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';

	let username = $state('');
	let password = $state('');
	let submitting = $state(false);
	let errorMessage = $state('');

	function describeError(err: unknown): string {
		if (err instanceof ApiError) {
			switch (err.status) {
				case 401:
					return err.message || 'Incorrect username or password.';
				case 403:
					return 'You signed in, but your account is not mapped to a role. Contact an administrator.';
				case 503:
					return 'The directory is unreachable. Try again shortly, or sign in with a local account.';
				default:
					return err.message || 'Sign-in failed.';
			}
		}
		return 'The gateway could not be reached. Check that it is running and try again.';
	}

	async function handleSubmit(event: SubmitEvent) {
		event.preventDefault();
		errorMessage = '';
		submitting = true;
		try {
			await session.login(username, password);
			password = '';
			goto('/');
		} catch (err) {
			errorMessage = describeError(err);
		} finally {
			submitting = false;
		}
	}
</script>

<div class="flex min-h-screen flex-col items-center justify-center gap-6 px-4">
	<form class="w-full max-w-sm border" style="background:var(--color-base-100); border-color:var(--color-base-300); border-radius:4px" onsubmit={handleSubmit} data-testid="login-form">
		<div class="grid gap-4 p-6">
			<div>
				<h1 class="text-2xl font-bold">Signal Gateway</h1>
				<p class="muted text-[15px]">Sign in to manage numbers and devices.</p>
			</div>

			{#if errorMessage}
				<div class="alert alert-error text-sm" role="alert" data-testid="login-error">{errorMessage}</div>
			{/if}

			<div>
				<label class="field-label" for="login-username">Username</label>
				<input id="login-username" class="input w-full" type="text" name="username" autocomplete="username" bind:value={username} required />
			</div>
			<div>
				<label class="field-label" for="login-password">Password</label>
				<input id="login-password" class="input w-full" type="password" name="password" autocomplete="current-password" bind:value={password} required />
			</div>
			<button type="submit" class="xbtn xbtn-solid xbtn-lg" data-testid="login-submit" disabled={submitting}>
				{submitting ? 'Signing in…' : 'Sign in'}
			</button>
		</div>
	</form>
	<ThemeToggle />
</div>
