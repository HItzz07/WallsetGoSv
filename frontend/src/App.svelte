<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import {
    GetWallpapers,
    GetSettings,
    UpdateSettings,
    DownloadAndSetWallpaper,
    SetWallpaper,
    DeleteWallpaper,
    GetWallpaperDirectory,
    OpenWallpaperDirectory
  } from '../wailsjs/go/main/App';
  import { EventsOn } from '../wailsjs/runtime';

  interface WallpaperInfo {
    id: string;
    filename: string;
    filepath: string;
    local_url: string;
    download_date: string;
    source_url: string;
    file_size: number;
  }

  interface AppSettings {
    auto_change_enabled: boolean;
    change_interval_hours: number;
    download_sources: string[];
    max_wallpapers: number;
  }

  let wallpapers: WallpaperInfo[] = [];
  let settings: AppSettings | null = null;
  let currentTab = 'wallpapers';
  let isLoading = false;
  let status = 'Ready';
  let wallpaperDirectory = '';
  let sourcesText = '';

  // Event cleanup functions
  let unsubscribeWallpaperChanged: (() => void) | null = null;
  let unsubscribeWallpapersUpdated: (() => void) | null = null;

  onMount(async () => {
    await loadData();
    
    // Set up event listeners
    unsubscribeWallpaperChanged = EventsOn('wallpaperChanged', async (info: WallpaperInfo) => {
      await loadWallpapers();
      status = `New wallpaper set: ${info.filename}`;
    });

    unsubscribeWallpapersUpdated = EventsOn('wallpapersUpdated', async () => {
      await loadWallpapers();
      status = 'Wallpapers updated';
    });
  });

  onDestroy(() => {
    if (unsubscribeWallpaperChanged) unsubscribeWallpaperChanged();
    if (unsubscribeWallpapersUpdated) unsubscribeWallpapersUpdated();
  });

  async function loadData() {
    await loadWallpapers();
    await loadSettings();
    wallpaperDirectory = await GetWallpaperDirectory();
  }

  async function loadWallpapers() {
    try {
      wallpapers = await GetWallpapers();
    } catch (err) {
      console.error('Failed to load wallpapers:', err);
      status = `Error loading wallpapers: ${err}`;
    }
  }

  async function loadSettings() {
    try {
      settings = await GetSettings();
      if (settings) {
        sourcesText = settings.download_sources.join('\n');
      }
    } catch (err) {
      console.error('Failed to load settings:', err);
      status = `Error loading settings: ${err}`;
    }
  }

  async function handleDownload() {
    if (isLoading) return;
    
    isLoading = true;
    status = 'Downloading new wallpaper...';
    
    try {
      const info = await DownloadAndSetWallpaper();
      if (info) {
        await loadWallpapers();
        status = `Downloaded and set: ${info.filename} (${(info.file_size / 1024 / 1024).toFixed(1)}MB)`;
      }
    } catch (err) {
      status = `Download failed: ${err}`;
    } finally {
      isLoading = false;
    }
  }

  async function handleSet(filepath: string, filename: string) {
    status = `Setting wallpaper: ${filename}`;
    try {
      await SetWallpaper(filepath);
      status = `Wallpaper set: ${filename}`;
    } catch (err) {
      status = `Error setting wallpaper: ${err}`;
    }
  }

  async function handleDelete(id: string, filename: string) {
    if (confirm(`Delete ${filename}?`)) {
      try {
        await DeleteWallpaper(id);
        status = `Deleted: ${filename}`;
      } catch (err) {
        status = `Error deleting: ${err}`;
      }
    }
  }

  async function handleSaveSettings() {
    if (!settings) return;
    
    // Update sources from text area
    settings.download_sources = sourcesText.split('\n').filter(s => s.trim() !== '');
    
    try {
      await UpdateSettings(settings);
      status = 'Settings saved successfully';
    } catch (err) {
      status = `Error saving settings: ${err}`;
    }
  }

  async function handleOpenDirectory() {
    try {
      await OpenWallpaperDirectory();
      status = 'Opened wallpaper directory';
    } catch (err) {
      status = `Error opening directory: ${err}`;
    }
  }

  // Fixed image error handler with proper typing
  function handleImageError(event: Event) {
    const target = event.currentTarget as HTMLImageElement;
    if (target) {
      target.src = 'data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjAwIiBoZWlnaHQ9IjEwMCIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48ZGVmcz48L2RlZnM+PGcgZmlsbD0ibm9uZSI+PHJlY3Qgd2lkdGg9IjIwMCIgaGVpZ2h0PSIxMDAiIGZpbGw9IiNjY2MiLz48dGV4dCB4PSIxMDAiIHk9IjUwIiBmb250LXNpemU9IjE0IiBmaWxsPSIjMzMzIiB0ZXh0LWFuY2hvcj0ibWlkZGxlIiBkeT0iLjNlbSI+SW1hZ2UgTm90IEF2YWlsYWJsZTwvdGV4dD48L2c+PC9zdmc+';
    }
  }

  function formatFileSize(bytes: number): string {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1048576) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / 1048576).toFixed(1) + ' MB';
  }

  function formatDate(dateStr: string): string {
    return new Date(dateStr).toLocaleString();
  }
</script>

<main data-theme="dark" class="bg-base-200 min-h-screen flex flex-col">
  <!-- Header -->
  <div class="p-4 bg-base-300 shadow-lg">
    <div class="flex justify-between items-center mb-2">
      <h1 class="text-2xl font-bold flex items-center gap-2">
        🖼️ Wallpaper Engine
      </h1>
      <div class="flex gap-2">
        <button class="btn btn-ghost btn-sm" on:click={handleOpenDirectory} title="Open Folder">
          📁 Open Folder
        </button>
        <button class="btn btn-primary" on:click={handleDownload} disabled={isLoading}>
          {#if isLoading}
            <span class="loading loading-spinner loading-sm"></span>
          {/if}
          🆕 Download New
        </button>
      </div>
    </div>
    <div class="text-sm opacity-70">
      <span>{status}</span>
      {#if wallpaperDirectory}
        <span class="ml-4">📁 {wallpaperDirectory}</span>
      {/if}
    </div>
  </div>

  <!-- Tabs -->
  <div role="tablist" class="tabs tabs-boxed rounded-none bg-base-300 px-4">
    <button 
      role="tab" 
      class="tab" 
      class:tab-active={currentTab === 'wallpapers'}
      on:click={() => currentTab = 'wallpapers'}
    >
      🖼️ Wallpapers ({wallpapers.length})
    </button>
    <button 
      role="tab" 
      class="tab" 
      class:tab-active={currentTab === 'settings'}
      on:click={() => currentTab = 'settings'}
    >
      ⚙️ Settings
    </button>
  </div>

  <!-- Content -->
  <div class="p-4 overflow-y-auto flex-grow">
    {#if currentTab === 'wallpapers'}
      {#if wallpapers.length === 0}
        <div class="text-center py-20">
          <p class="text-lg opacity-70 mb-4">No wallpapers yet</p>
          <button class="btn btn-primary" on:click={handleDownload} disabled={isLoading}>
            Download Your First Wallpaper
          </button>
        </div>
      {:else}
        <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
          {#each wallpapers as wallpaper}
            <div class="card bg-base-100 shadow-xl">
              <figure class="aspect-video bg-base-200">
                <img 
                  src={wallpaper.local_url} 
                  alt="Wallpaper preview"
                  class="object-cover w-full h-full"
                  loading="lazy"
                  on:error={handleImageError}
                />
              </figure>
              <div class="card-body p-3">
                <div class="text-xs opacity-70 mb-2">
                  {formatDate(wallpaper.download_date)}<br>
                  {formatFileSize(wallpaper.file_size)}
                </div>
                <div class="card-actions justify-between">
                  <button 
                    class="btn btn-primary btn-sm flex-1 mr-1" 
                    on:click={() => handleSet(wallpaper.filepath, wallpaper.filename)}
                  >
                    🎯 Set
                  </button>
                  <button 
                    class="btn btn-error btn-sm" 
                    on:click={() => handleDelete(wallpaper.id, wallpaper.filename)}
                    title="Delete"
                  >
                    🗑️
                  </button>
                </div>
              </div>
            </div>
          {/each}
        </div>
      {/if}

    {:else if currentTab === 'settings' && settings}
      <div class="max-w-2xl mx-auto space-y-6">
        <div class="card bg-base-100 shadow-xl">
          <div class="card-body">
            <h2 class="card-title">⚙️ Auto-Change Settings</h2>
            
            <div class="form-control">
              <label class="label cursor-pointer">
                <span class="label-text">Enable automatic wallpaper change</span>
                <input 
                  id="auto-change-toggle"
                  type="checkbox" 
                  class="toggle toggle-primary" 
                  bind:checked={settings.auto_change_enabled} 
                />
              </label>
            </div>

            <div class="form-control">
              <label class="label" for="interval-input">
                <span class="label-text">Change interval (hours)</span>
              </label>
              <input 
                id="interval-input"
                type="number" 
                class="input input-bordered" 
                min="1" 
                max="24" 
                bind:value={settings.change_interval_hours} 
              />
            </div>

            <div class="form-control">
              <label class="label" for="max-wallpapers-input">
                <span class="label-text">Maximum wallpapers to keep</span>
              </label>
              <input 
                id="max-wallpapers-input"
                type="number" 
                class="input input-bordered" 
                min="5" 
                max="100" 
                bind:value={settings.max_wallpapers} 
              />
            </div>
          </div>
        </div>

        <div class="card bg-base-100 shadow-xl">
          <div class="card-body">
            <h2 class="card-title">🌐 Download Sources</h2>
            <p class="text-sm opacity-70 mb-4">
              High-quality 2K/4K wallpaper sources (one per line):
            </p>
            
            <div class="form-control">
              <label class="label" for="sources-textarea">
                <span class="label-text">Download sources URLs</span>
              </label>
              <textarea 
                id="sources-textarea"
                class="textarea textarea-bordered h-48 font-mono text-sm" 
                placeholder="https://source.unsplash.com/3840x2160/landscape&#10;https://source.unsplash.com/3840x2160/nature&#10;https://picsum.photos/3840/2160"
                bind:value={sourcesText}
              ></textarea>
            </div>
            
            <div class="alert alert-info mt-4">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" class="stroke-current shrink-0 w-6 h-6">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
              </svg>
              <div class="text-sm">
                <strong>Current sources include:</strong><br>
                • 4K Unsplash images (3840x2160)<br>
                • 2K Unsplash images (2560x1440)<br>
                • Picsum random high-quality images<br>
                • Various categories: landscape, nature, architecture, space
              </div>
            </div>
          </div>
        </div>

        <div class="card-actions justify-end">
          <button class="btn btn-primary btn-lg" on:click={handleSaveSettings}>
            💾 Save Settings
          </button>
        </div>
      </div>
    {/if}
  </div>
</main>

<style>
  :global(body) {
    margin: 0;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  }
</style>
