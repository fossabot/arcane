<script lang="ts">
	import { onMount } from 'svelte';
	import { z } from 'zod/v4';
	import settingsStore from '$lib/stores/config-store';
	import { m } from '$lib/paraglide/messages';
	import { SettingsPageLayout } from '$lib/layouts';
	import { Label } from '$lib/components/ui/label';
	import { ClockIcon } from '$lib/icons';
	import TextInputWithLabel from '$lib/components/form/text-input-with-label.svelte';
	import SelectWithLabel from '$lib/components/form/select-with-label.svelte';
	import { createSettingsForm } from '$lib/utils/settings-form.util';
	import { settingsService } from '$lib/services/settings-service';

	let { data } = $props();

	const currentSettings = $derived($settingsStore || data.settings!);
	const isReadOnly = $derived.by(() => $settingsStore?.uiConfigDisabled);

	const formSchema = z.object({
		dockerApiTimeout: z.coerce.number().int().min(1).max(3600),
		dockerImagePullTimeout: z.coerce.number().int().min(30).max(7200),
		gitOperationTimeout: z.coerce.number().int().min(30).max(3600),
		httpClientTimeout: z.coerce.number().int().min(5).max(300),
		registryTimeout: z.coerce.number().int().min(5).max(300),
		proxyRequestTimeout: z.coerce.number().int().min(10).max(600),
		buildProvider: z.enum(['local', 'depot']).default('local'),
		buildkitEndpoint: z.string().default(''),
		buildTimeout: z.coerce.number().int().min(60).max(14400),
		depotProjectId: z.string().default(''),
		depotToken: z.string().optional().default('')
	});

	const formDefaults = $derived({
		dockerApiTimeout: currentSettings.dockerApiTimeout,
		dockerImagePullTimeout: currentSettings.dockerImagePullTimeout,
		gitOperationTimeout: currentSettings.gitOperationTimeout,
		httpClientTimeout: currentSettings.httpClientTimeout,
		registryTimeout: currentSettings.registryTimeout,
		proxyRequestTimeout: currentSettings.proxyRequestTimeout,
		buildProvider: currentSettings.buildProvider,
		buildkitEndpoint: currentSettings.buildkitEndpoint,
		buildTimeout: currentSettings.buildTimeout,
		depotProjectId: currentSettings.depotProjectId,
		depotToken: ''
	});

	let { formInputs, registerOnMount } = $derived(
		createSettingsForm({
			schema: formSchema,
			currentSettings: formDefaults,
			getCurrentSettings: () => ({
				dockerApiTimeout: ($settingsStore || data.settings!).dockerApiTimeout,
				dockerImagePullTimeout: ($settingsStore || data.settings!).dockerImagePullTimeout,
				gitOperationTimeout: ($settingsStore || data.settings!).gitOperationTimeout,
				httpClientTimeout: ($settingsStore || data.settings!).httpClientTimeout,
				registryTimeout: ($settingsStore || data.settings!).registryTimeout,
				proxyRequestTimeout: ($settingsStore || data.settings!).proxyRequestTimeout,
				buildProvider: ($settingsStore || data.settings!).buildProvider,
				buildkitEndpoint: ($settingsStore || data.settings!).buildkitEndpoint,
				buildTimeout: ($settingsStore || data.settings!).buildTimeout,
				depotProjectId: ($settingsStore || data.settings!).depotProjectId,
				depotToken: ''
			}),
			onSave: async (payload) => {
				const updated = { ...payload } as Record<string, unknown>;
				if (!updated.depotToken) {
					delete updated.depotToken;
				}
				await settingsService.updateSettings(updated);
			},
			successMessage: m.timeouts_save()
		})
	);

	onMount(() => registerOnMount());
</script>

<SettingsPageLayout
	title={m.timeouts_settings()}
	description={m.timeouts_settings_description()}
	icon={ClockIcon}
	pageType="form"
	showReadOnlyTag={isReadOnly}
>
	{#snippet mainContent()}
		<fieldset disabled={isReadOnly} class="relative space-y-8">
			<!-- Docker Operations -->
			<div class="space-y-4">
				<h3 class="text-lg font-medium">Docker Operations</h3>
				<div class="bg-card rounded-lg border shadow-sm">
					<div class="space-y-6 p-6">
						<div class="grid gap-4 md:grid-cols-[1fr_1.5fr] md:gap-8">
							<div>
								<Label class="text-base">{m.docker_api_timeout()}</Label>
								<p class="text-muted-foreground mt-1 text-sm">
									{m.docker_api_timeout_description()}
								</p>
							</div>
							<div class="max-w-xs">
								<TextInputWithLabel
									bind:value={$formInputs.dockerApiTimeout.value}
									error={$formInputs.dockerApiTimeout.error}
									label={m.docker_api_timeout()}
									placeholder="30"
									helpText="Timeout in seconds (1-3600)"
									type="number"
								/>
							</div>
						</div>

						<div class="border-t pt-6">
							<div class="grid gap-4 md:grid-cols-[1fr_1.5fr] md:gap-8">
								<div>
									<Label class="text-base">{m.docker_image_pull_timeout()}</Label>
									<p class="text-muted-foreground mt-1 text-sm">
										{m.docker_image_pull_timeout_description()}
									</p>
								</div>
								<div class="max-w-xs">
									<TextInputWithLabel
										bind:value={$formInputs.dockerImagePullTimeout.value}
										error={$formInputs.dockerImagePullTimeout.error}
										label={m.docker_image_pull_timeout()}
										placeholder="600"
										helpText="Timeout in seconds (30-7200)"
										type="number"
									/>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Git Operations -->
			<div class="space-y-4">
				<h3 class="text-lg font-medium">Git Operations</h3>
				<div class="bg-card rounded-lg border shadow-sm">
					<div class="space-y-6 p-6">
						<div class="grid gap-4 md:grid-cols-[1fr_1.5fr] md:gap-8">
							<div>
								<Label class="text-base">{m.git_operation_timeout()}</Label>
								<p class="text-muted-foreground mt-1 text-sm">
									{m.git_operation_timeout_description()}
								</p>
							</div>
							<div class="max-w-xs">
								<TextInputWithLabel
									bind:value={$formInputs.gitOperationTimeout.value}
									error={$formInputs.gitOperationTimeout.error}
									label={m.git_operation_timeout()}
									placeholder="300"
									helpText="Timeout in seconds (30-3600)"
									type="number"
								/>
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Network Operations -->
			<div class="space-y-4">
				<h3 class="text-lg font-medium">Network Operations</h3>
				<div class="bg-card rounded-lg border shadow-sm">
					<div class="space-y-6 p-6">
						<div class="grid gap-4 md:grid-cols-[1fr_1.5fr] md:gap-8">
							<div>
								<Label class="text-base">{m.http_client_timeout()}</Label>
								<p class="text-muted-foreground mt-1 text-sm">
									{m.http_client_timeout_description()}
								</p>
							</div>
							<div class="max-w-xs">
								<TextInputWithLabel
									bind:value={$formInputs.httpClientTimeout.value}
									error={$formInputs.httpClientTimeout.error}
									label={m.http_client_timeout()}
									placeholder="30"
									helpText="Timeout in seconds (5-300)"
									type="number"
								/>
							</div>
						</div>

						<div class="border-t pt-6">
							<div class="grid gap-4 md:grid-cols-[1fr_1.5fr] md:gap-8">
								<div>
									<Label class="text-base">{m.registry_timeout()}</Label>
									<p class="text-muted-foreground mt-1 text-sm">
										{m.registry_timeout_description()}
									</p>
								</div>
								<div class="max-w-xs">
									<TextInputWithLabel
										bind:value={$formInputs.registryTimeout.value}
										error={$formInputs.registryTimeout.error}
										label={m.registry_timeout()}
										placeholder="30"
										helpText="Timeout in seconds (5-300)"
										type="number"
									/>
								</div>
							</div>
						</div>

						<div class="border-t pt-6">
							<div class="grid gap-4 md:grid-cols-[1fr_1.5fr] md:gap-8">
								<div>
									<Label class="text-base">{m.proxy_request_timeout()}</Label>
									<p class="text-muted-foreground mt-1 text-sm">
										{m.proxy_request_timeout_description()}
									</p>
								</div>
								<div class="max-w-xs">
									<TextInputWithLabel
										bind:value={$formInputs.proxyRequestTimeout.value}
										error={$formInputs.proxyRequestTimeout.error}
										label={m.proxy_request_timeout()}
										placeholder="60"
										helpText="Timeout in seconds (10-600)"
										type="number"
									/>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Build Operations -->
			<div class="space-y-4">
				<h3 class="text-lg font-medium">Build Operations</h3>
				<div class="bg-card rounded-lg border shadow-sm">
					<div class="space-y-6 p-6">
						<div class="grid gap-4 md:grid-cols-[1fr_1.5fr] md:gap-8">
							<div>
								<Label class="text-base">Build Provider</Label>
								<p class="text-muted-foreground mt-1 text-sm">Select the default BuildKit provider.</p>
							</div>
							<div class="max-w-xs">
								<SelectWithLabel
									id="build-provider"
									name="buildProvider"
									bind:value={$formInputs.buildProvider.value}
									error={$formInputs.buildProvider.error}
									label="Build Provider"
									options={[
										{ label: 'Local BuildKit', value: 'local', description: 'Use the local BuildKit daemon' },
										{ label: 'Depot', value: 'depot', description: 'Use Depot hosted BuildKit' }
									]}
								/>
							</div>
						</div>

						<div class="border-t pt-6">
							<div class="grid gap-4 md:grid-cols-[1fr_1.5fr] md:gap-8">
								<div>
									<Label class="text-base">BuildKit Endpoint</Label>
									<p class="text-muted-foreground mt-1 text-sm">
										BuildKit daemon address (e.g., unix:///run/buildkit/buildkitd.sock).
									</p>
								</div>
								<div class="max-w-xl">
									<TextInputWithLabel
										bind:value={$formInputs.buildkitEndpoint.value}
										error={$formInputs.buildkitEndpoint.error}
										label="BuildKit Endpoint"
										placeholder="unix:///run/buildkit/buildkitd.sock"
										helpText="Leave blank to use the default BuildKit socket"
									/>
								</div>
							</div>
						</div>

						<div class="border-t pt-6">
							<div class="grid gap-4 md:grid-cols-[1fr_1.5fr] md:gap-8">
								<div>
									<Label class="text-base">Build Timeout</Label>
									<p class="text-muted-foreground mt-1 text-sm">Timeout for image builds in seconds.</p>
								</div>
								<div class="max-w-xs">
									<TextInputWithLabel
										bind:value={$formInputs.buildTimeout.value}
										error={$formInputs.buildTimeout.error}
										label="Build Timeout"
										placeholder="1800"
										helpText="Timeout in seconds (60-14400)"
										type="number"
									/>
								</div>
							</div>
						</div>

						<div class="border-t pt-6">
							<div class="grid gap-4 md:grid-cols-[1fr_1.5fr] md:gap-8">
								<div>
									<Label class="text-base">Depot Project ID</Label>
									<p class="text-muted-foreground mt-1 text-sm">Depot project identifier for hosted builds.</p>
								</div>
								<div class="max-w-xl">
									<TextInputWithLabel
										bind:value={$formInputs.depotProjectId.value}
										error={$formInputs.depotProjectId.error}
										label="Depot Project ID"
										placeholder="proj_123456"
									/>
								</div>
							</div>
						</div>

						<div class="border-t pt-6">
							<div class="grid gap-4 md:grid-cols-[1fr_1.5fr] md:gap-8">
								<div>
									<Label class="text-base">Depot Token</Label>
									<p class="text-muted-foreground mt-1 text-sm">
										Personal access token for Depot (leave blank to keep existing).
									</p>
								</div>
								<div class="max-w-xl">
									<TextInputWithLabel
										bind:value={$formInputs.depotToken.value}
										error={$formInputs.depotToken.error}
										label="Depot Token"
										placeholder="******"
										type="password"
										helpText="Leave blank to preserve the existing token"
									/>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>
		</fieldset>
	{/snippet}
</SettingsPageLayout>
