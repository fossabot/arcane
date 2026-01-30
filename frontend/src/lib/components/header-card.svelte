<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '$lib/utils';

	interface Props {
		children: Snippet;
		class?: string;
	}

	let { children, class: className }: Props = $props();
</script>

<div
	class={cn(
		'from-background/60 via-background/40 to-background/60 backdrop-blur-2xs text-foreground dark:bg-surface/10 dark:border-primary/9 relative overflow-hidden rounded-xl border border-white/80 bg-white/10 bg-linear-to-br p-4 shadow-sm sm:p-6',
		className
	)}
>
	<div class="bg-primary/10 pointer-events-none absolute -top-10 -right-10 size-40 rounded-full blur-3xl"></div>
	<div class="bg-muted/40 pointer-events-none absolute -bottom-10 -left-10 size-40 rounded-full blur-3xl"></div>

	<div class="relative">
		{@render children()}
	</div>
</div>
