<script lang="ts">
	import * as ResponsiveDialog from '$lib/components/ui/responsive-dialog/index.js';
	import { ArcaneButton } from '$lib/components/arcane-button/index.js';
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Progress } from '$lib/components/ui/progress/index.js';
	import * as Collapsible from '$lib/components/ui/collapsible/index.js';
	import SwitchWithLabel from '$lib/components/form/labeled-switch.svelte';
	import SelectWithLabel from '$lib/components/form/select-with-label.svelte';
	import { z } from 'zod/v4';
	import { createForm, preventDefault } from '$lib/utils/form.utils';
	import { toast } from 'svelte-sonner';
	import { environmentStore } from '$lib/stores/environment.store.svelte';
	import settingsStore from '$lib/stores/config-store';
	import { m } from '$lib/paraglide/messages';
	import { cn } from '$lib/utils.js';
	import {
		type LayerProgress,
		type PullPhase,
		calculateOverallProgress,
		areAllLayersComplete,
		updateLayerFromStreamData,
		extractErrorMessage,
		getLayerStats,
		getPullPhase,
		showImageLayersState,
		isIndeterminatePhase,
		getAggregateStatus
	} from '$lib/utils/pull-progress';
	import { ArrowDownIcon, DownloadIcon } from '$lib/icons';

	type ImageBuildFormProps = {
		open: boolean;
		onBuildFinished?: (success: boolean, tags?: string[], error?: string) => void;
	};

	let { open = $bindable(false), onBuildFinished = () => {} }: ImageBuildFormProps = $props();

	const defaultProvider = $derived((($settingsStore?.buildProvider as 'local' | 'depot') ?? 'local') as 'local' | 'depot');

	const formSchema = z.object({
		contextDir: z.string().min(1, 'Build context is required'),
		dockerfile: z.string().optional().default(''),
		tags: z.string().min(1, 'At least one tag is required'),
		target: z.string().optional().default(''),
		buildArgs: z.string().optional().default(''),
		platforms: z.string().optional().default(''),
		provider: z.enum(['local', 'depot']).default('local'),
		push: z.boolean().default(false),
		load: z.boolean().default(true)
	});

	const { inputs, ...form } = createForm<typeof formSchema>(formSchema, {
		contextDir: '',
		dockerfile: '',
		tags: '',
		target: '',
		buildArgs: '',
		platforms: '',
		provider: 'local',
		push: false,
		load: true
	});

	let isBuilding = $state(false);
	let buildProgress = $state(0);
	let buildStatusText = $state('');
	let buildError = $state('');
	let layerProgress = $state<Record<string, LayerProgress>>({});
	let hasReachedComplete = $state(false);
	let currentTags = $state<string[]>([]);
	const layerStats = $derived(getLayerStats(layerProgress, hasReachedComplete));
	const aggregateStatus = $derived(getAggregateStatus(layerProgress, buildStatusText, hasReachedComplete));
	const showBuildUI = $derived(isBuilding || hasReachedComplete || !!buildError);
	const isIndeterminate = $derived(isIndeterminatePhase(layerProgress, buildProgress));
	let prevOpen = $state(false);
	let showAdvanced = $state(false);

	$effect(() => {
		if (prevOpen && !open && !isBuilding) {
			resetState();
			$inputs.contextDir.value = '';
			$inputs.tags.value = '';
			$inputs.dockerfile.value = '';
			$inputs.target.value = '';
			$inputs.buildArgs.value = '';
			$inputs.platforms.value = '';
		}
		prevOpen = open;
	});

	$effect(() => {
		if ($inputs.provider.value === 'depot') {
			$inputs.push.value = true;
			$inputs.load.value = false;
		}
	});

	function getLayerPhase(status: string): PullPhase {
		return getPullPhase(status, false, false);
	}

	function resetState() {
		isBuilding = false;
		buildProgress = 0;
		buildStatusText = '';
		buildError = '';
		layerProgress = {};
		hasReachedComplete = false;
		currentTags = [];
	}

	function updateProgress() {
		buildProgress = calculateOverallProgress(layerProgress);
	}

	function parseTags(raw: string): string[] {
		return raw
			.split(/[,\n]/)
			.map((t) => t.trim())
			.filter(Boolean);
	}

	function parsePlatforms(raw: string): string[] {
		return raw
			.split(/[,\n]/)
			.map((t) => t.trim())
			.filter(Boolean);
	}

	function parseBuildArgs(raw: string): Record<string, string> {
		const result: Record<string, string> = {};
		for (const line of raw.split('\n')) {
			const trimmed = line.trim();
			if (!trimmed) continue;
			const idx = trimmed.indexOf('=');
			if (idx === -1) continue;
			const key = trimmed.slice(0, idx).trim();
			const value = trimmed.slice(idx + 1).trim();
			if (!key) continue;
			result[key] = value;
		}
		return result;
	}

	async function handleSubmit() {
		const data = form.validate();
		if (!data) return;

		resetState();
		isBuilding = true;
		buildStatusText = 'Starting build...';

		const tags = parseTags(data.tags);
		currentTags = tags;

		const payload = {
			contextDir: data.contextDir.trim(),
			dockerfile: data.dockerfile?.trim() || undefined,
			tags,
			target: data.target?.trim() || undefined,
			buildArgs: parseBuildArgs(data.buildArgs || ''),
			platforms: parsePlatforms(data.platforms || ''),
			provider: data.provider,
			push: data.push,
			load: data.load
		};

		try {
			const envId = await environmentStore.getCurrentEnvironmentId();
			const response = await fetch(`/api/environments/${envId}/images/build`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify(payload)
			});

			if (!response.ok || !response.body) {
				const errorData = await response.json().catch(() => ({
					data: { message: 'Build request failed' }
				}));

				const errorMessage =
					errorData.data?.message || errorData.error || errorData.message || `Build request failed: HTTP ${response.status}`;

				throw new Error(errorMessage);
			}

			const reader = response.body.getReader();
			const decoder = new TextDecoder();
			let buffer = '';

			while (true) {
				const { done, value } = await reader.read();
				if (done) {
					buildStatusText = 'Finalizing build...';
					break;
				}

				buffer += decoder.decode(value, { stream: true });
				const lines = buffer.split('\n');
				buffer = lines.pop() || '';

				for (const line of lines) {
					if (line.trim() === '') continue;
					try {
						const event = JSON.parse(line);

						const errorMsg = extractErrorMessage(event, 'Build failed');
						if (errorMsg) {
							buildError = errorMsg;
							buildStatusText = `Build failed: ${errorMsg}`;
							continue;
						}

						if (event.status) buildStatusText = event.status;
						layerProgress = updateLayerFromStreamData(layerProgress, event);
						updateProgress();
					} catch (e: any) {
						console.warn('Failed to parse build stream line:', line, e);
					}
				}
			}

			updateProgress();
			if (!buildError && buildProgress < 100 && areAllLayersComplete(layerProgress)) {
				buildProgress = 100;
			}

			if (buildError) {
				throw new Error(buildError);
			}

			hasReachedComplete = true;
			buildProgress = 100;
			buildStatusText = 'Build completed';
			toast.success('Build completed');
			onBuildFinished(true, tags);

			setTimeout(() => {
				open = false;
			}, 1500);
		} catch (error: any) {
			const message = error.message || 'Build failed';
			buildError = message;
			buildStatusText = `Build failed: ${message}`;
			toast.error(message);
			onBuildFinished(false, tags, message);
		} finally {
			isBuilding = false;
		}
	}

	function handleOpenChange(newOpenState: boolean) {
		if (!newOpenState && isBuilding) {
			toast.info('Build is still running');
			open = true;
			return;
		}

		open = newOpenState;
		if (!newOpenState) {
			resetState();
			$inputs.contextDir.value = '';
			$inputs.tags.value = '';
			$inputs.dockerfile.value = '';
			$inputs.target.value = '';
			$inputs.buildArgs.value = '';
			$inputs.platforms.value = '';
		} else {
			resetState();
			$inputs.provider.value = defaultProvider;
		}
	}
</script>

<ResponsiveDialog.Root
	bind:open
	onOpenChange={handleOpenChange}
	variant="sheet"
	title="Build Image"
	description={showBuildUI && currentTags.length > 0 ? currentTags.join(', ') : 'Build a Docker image with BuildKit'}
	contentClass="sm:max-w-[600px]"
>
	{#snippet children()}
		{#if showBuildUI}
			<div class="space-y-4 py-6">
				{#if buildError}
					<div class="bg-destructive/10 text-destructive rounded-lg p-4">
						<p class="text-sm font-medium">Build failed</p>
						<p class="mt-1 text-xs">{buildError}</p>
					</div>
				{:else}
					<div class="space-y-2">
						<div class="flex items-center justify-between">
							<p class="text-sm font-medium">
								{#if hasReachedComplete}
									Build completed
								{:else if layerStats.total > 0}
									<span class="flex items-center gap-1.5">
										<span>{aggregateStatus}</span>
										<span class="text-muted-foreground font-normal">•</span>
										<span class="text-muted-foreground font-normal">
											{m.progress_layers_status({ completed: layerStats.completed, total: layerStats.total })}
										</span>
									</span>
								{:else}
									{aggregateStatus || 'Building'}
								{/if}
							</p>
							{#if !isIndeterminate || hasReachedComplete}
								<p class="text-muted-foreground text-sm">{Math.round(hasReachedComplete ? 100 : buildProgress)}%</p>
							{/if}
						</div>
						<Progress
							value={hasReachedComplete || isIndeterminate ? 100 : buildProgress}
							max={100}
							class="h-2 w-full"
							indeterminate={isIndeterminate && !hasReachedComplete}
						/>
					</div>

					{#if Object.keys(layerProgress).length > 0}
						<Collapsible.Root bind:open={showImageLayersState.current}>
							<Collapsible.Trigger
								class="text-muted-foreground hover:text-foreground hover:bg-accent flex w-full items-center justify-between rounded-md px-2 py-1.5 text-xs transition-colors"
							>
								{m.progress_show_layers()}
								<ArrowDownIcon class={cn('size-4 transition-transform', showImageLayersState.current && 'rotate-180')} />
							</Collapsible.Trigger>
							<Collapsible.Content>
								<div class="mt-2 space-y-1.5">
									{#each Object.entries(layerProgress) as [id, layer] (id)}
										{@const phase = hasReachedComplete ? 'complete' : getLayerPhase(layer.status)}
										{@const layerPercent =
											phase === 'complete' ? 100 : layer.total > 0 ? Math.round((layer.current / layer.total) * 100) : 0}
										<div class="bg-muted/30 rounded-md px-2 py-1.5">
											<div class="flex items-center justify-between gap-2">
												<span class="text-muted-foreground truncate font-mono text-[10px]">{id.slice(0, 12)}</span>
												<span
													class={cn(
														'shrink-0 text-[10px] font-medium',
														phase === 'complete' && 'text-green-500',
														phase === 'downloading' && 'text-blue-500',
														phase === 'extracting' && 'text-amber-500'
													)}
												>
													{#if phase === 'complete'}
														✓
													{:else if layer.total > 0}
														{layerPercent}%
													{:else}
														{layer.status}
													{/if}
												</span>
											</div>
											<Progress value={layerPercent} max={100} class="mt-1 h-1" />
										</div>
									{/each}
								</div>
							</Collapsible.Content>
						</Collapsible.Root>
					{/if}

					{#if isBuilding}
						<p class="text-muted-foreground text-xs">{buildStatusText || 'Build in progress...'}</p>
					{/if}
				{/if}
			</div>
		{:else}
			<form onsubmit={preventDefault(handleSubmit)} class="grid gap-4 py-6">
				<FormInput
					label="Context Directory"
					type="text"
					placeholder="/app/data/projects/my-project"
					description="Path to the build context on the server"
					bind:input={$inputs.contextDir}
				/>

				<FormInput
					label="Image Tags"
					type="text"
					placeholder="my-image:latest"
					description="Comma-separated list of tags"
					bind:input={$inputs.tags}
				/>

				<SelectWithLabel
					id="build-provider"
					name="buildProvider"
					bind:value={$inputs.provider.value}
					label="Build Provider"
					options={[
						{ label: 'Local BuildKit', value: 'local', description: 'Use the local BuildKit daemon' },
						{ label: 'Depot', value: 'depot', description: 'Use Depot hosted BuildKit' }
					]}
				/>

				<div class="grid gap-4 sm:grid-cols-2">
					<SwitchWithLabel
						id="build-push"
						checked={$inputs.push.value}
						label="Push"
						description="Push built images to registry"
						onCheckedChange={(v) => ($inputs.push.value = v)}
					/>
					<SwitchWithLabel
						id="build-load"
						checked={$inputs.load.value}
						label="Load"
						description="Load image into local Docker"
						onCheckedChange={(v) => ($inputs.load.value = v)}
						disabled={$inputs.provider.value === 'depot'}
					/>
				</div>

				<Collapsible.Root bind:open={showAdvanced}>
					<Collapsible.Trigger
						class="text-muted-foreground hover:text-foreground hover:bg-accent flex w-full items-center justify-between rounded-md px-2 py-1.5 text-xs transition-colors"
					>
						Advanced options
						<ArrowDownIcon class={cn('size-4 transition-transform', showAdvanced && 'rotate-180')} />
					</Collapsible.Trigger>
					<Collapsible.Content>
						<div class="mt-4 grid gap-4">
							<FormInput
								label="Dockerfile"
								type="text"
								placeholder="Dockerfile"
								description="Path to Dockerfile (relative to context)"
								bind:input={$inputs.dockerfile}
							/>
							<FormInput
								label="Target"
								type="text"
								placeholder="builder"
								description="Target stage in the Dockerfile"
								bind:input={$inputs.target}
							/>
							<FormInput
								label="Build Args"
								type="textarea"
								rows={4}
								placeholder="KEY=value"
								description="One KEY=value per line"
								bind:input={$inputs.buildArgs}
							/>
							<FormInput
								label="Platforms"
								type="text"
								placeholder="linux/amd64, linux/arm64"
								description="Comma-separated list of platforms"
								bind:input={$inputs.platforms}
							/>
						</div>
					</Collapsible.Content>
				</Collapsible.Root>
			</form>
		{/if}
	{/snippet}

	{#snippet footer()}
		{#if buildError}
			<div class="flex w-full flex-row gap-2">
				<ArcaneButton
					action="base"
					tone="outline"
					type="button"
					class="flex-1"
					onclick={() => resetState()}
					customLabel={m.common_retry()}
				/>
				<ArcaneButton action="base" type="button" class="flex-1" onclick={() => (open = false)} customLabel={m.common_close()} />
			</div>
		{:else if !showBuildUI}
			<div class="flex w-full flex-row gap-2">
				<ArcaneButton action="cancel" tone="outline" type="button" class="flex-1" onclick={() => (open = false)} />
				<ArcaneButton action="base" type="submit" class="flex-1" onclick={handleSubmit} customLabel="Build" icon={DownloadIcon} />
			</div>
		{/if}
	{/snippet}
</ResponsiveDialog.Root>
