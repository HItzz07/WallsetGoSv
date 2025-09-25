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
    OpenWallpaperDirectory,
    GetWallpaperAsBase64
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
  let currentTab = 'download'; // download, gallery, settings
  let isLoading = false;
  let status = 'Ready';
  let wallpaperDirectory = '';
  let sourcesText = '';
  let imageCache: Map<string, string> = new Map();

  // Event cleanup functions
  let unsubscribeWallpaperChanged: (() => void) | null = null;
  let unsubscribeWallpapersUpdated: (() => void) | null = null;

  onMount(async () => {
    await loadData();
    
    unsubscribeWallpaperChanged = EventsOn('wallpaperChanged', async (info: WallpaperInfo) => {
      await loadWallpapers();
      status = `✅ New wallpaper set: ${info.filename}`;
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
      for (let i = 0; i < Math.min(wallpapers.length, 4); i++) {
        loadImagePreview(wallpapers[i]);
      }
    } catch (err) {
      console.error('Failed to load wallpapers:', err);
      status = `❌ Error loading wallpapers: ${err}`;
    }
  }

  async function loadImagePreview(wallpaper: WallpaperInfo) {
    if (imageCache.has(wallpaper.id)) return;
    
    try {
      const base64 = await GetWallpaperAsBase64(wallpaper.filepath);
      imageCache.set(wallpaper.id, base64);
      imageCache = imageCache;
    } catch (err) {
      console.error('Failed to load image preview:', err);
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
      status = `❌ Error loading settings: ${err}`;
    }
  }

  async function handleDownload() {
    if (isLoading) return;
    
    isLoading = true;
    status = '⏳ Downloading wallpaper...';
    
    try {
      const info = await DownloadAndSetWallpaper();
      if (info) {
        await loadWallpapers();
        status = `✅ Downloaded and set: ${info.filename} (${(info.file_size / 1024 / 1024).toFixed(1)}MB)`;
      }
    } catch (err) {
      status = `❌ Download failed: ${err}`;
    } finally {
      isLoading = false;
    }
  }

  async function handleSet(filepath: string, filename: string) {
    status = `⚙️ Setting wallpaper: ${filename}`;
    try {
      await SetWallpaper(filepath);
      status = `✅ Wallpaper set: ${filename}`;
    } catch (err) {
      status = `❌ Error setting wallpaper: ${err}`;
    }
  }

  async function handleDelete(id: string, filename: string) {
    if (confirm(`🗑️ Delete ${filename}?`)) {
      try {
        await DeleteWallpaper(id);
        imageCache.delete(id);
        imageCache = imageCache;
        status = `🗑️ Deleted: ${filename}`;
      } catch (err) {
        status = `❌ Error deleting: ${err}`;
      }
    }
  }

  async function handleSaveSettings() {
    if (!settings) return;
    
    settings.download_sources = sourcesText.split('\n').filter(s => s.trim() !== '');
    
    try {
      await UpdateSettings(settings);
      status = '✅ Settings saved successfully';
    } catch (err) {
      status = `❌ Error saving settings: ${err}`;
    }
  }

  async function handleOpenDirectory() {
    try {
      await OpenWallpaperDirectory();
      status = '📁 Opened wallpaper directory';
    } catch (err) {
      status = `❌ Error opening directory: ${err}`;
    }
  }

  async function handleImageVisible(wallpaper: WallpaperInfo) {
    if (!imageCache.has(wallpaper.id)) {
      await loadImagePreview(wallpaper);
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

$: currentWallpaper = wallpapers.length > 0 
  ? wallpapers.reduce((latest, current) => 
      new Date(current.download_date) > new Date(latest.download_date) ? current : latest
    )
  : null;
</script>

<main data-theme="dark" class="bg-base-200 min-h-screen flex flex-col">
  <!-- Status Bar
  <div class="bg-base-300 px-4 py-2 text-sm">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2">
        <div class="badge badge-ghost text-xs">
          {status}
        </div>
      </div>
      {#if wallpaperDirectory}
        <div class="text-xs opacity-70">
          📁 {wallpaperDirectory}
        </div>
      {/if}
    </div>
  </div> -->

  <!-- Tab Navigation -->
  <div class="tabs tabs-boxed bg-base-300 px-4 py-2 rounded-none">
    <button 
      class="tab tab-lg" 
      class:tab-active={currentTab === 'download'}
      on:click={() => currentTab = 'download'}
    >
      <span class="flex items-center gap-2">
        🆕 <span class="hidden sm:inline">Download</span>
      </span>
    </button>
    <button 
      class="tab tab-lg" 
      class:tab-active={currentTab === 'gallery'}
      on:click={() => currentTab = 'gallery'}
    >
      <span class="flex items-center gap-2">
        🖼️ <span class="hidden sm:inline">Gallery ({wallpapers.length})</span>
        <!-- <div class="badge badge-sm badge-primary">{wallpapers.length}</div> -->
      </span>
    </button>
    <button 
      class="tab tab-lg" 
      class:tab-active={currentTab === 'settings'}
      on:click={() => currentTab = 'settings'}
    >
      <span class="flex items-center gap-2">
        ⚙️ <span class="hidden sm:inline">Settings</span>
      </span>
    </button>
  </div>

  <!-- Tab Content -->
  <div class="flex-grow overflow-y-auto">
    
    <!-- Download Tab -->
    {#if currentTab === 'download'}
      <div class="p-6">
        <div class="max-w-4xl mx-auto space-y-8">
          
          <!-- Download Section -->
          <div class="card bg-base-100 shadow-xl">
            <div class="card-body">
              <!-- <h2 class="card-title text-2xl">🆕 Get New Wallpaper</h2> -->
              <!-- <p class="text-base-content/70">Download and automatically set a high-quality 2K/4K wallpaper</p> -->
              
              <div class="card-actions justify-center pt-6">
                <button 
                  class="btn btn-primary btn-lg" 
                  class:loading={isLoading}
                  disabled={isLoading}
                  on:click={handleDownload}
                >
                  {#if isLoading}
                    <span class="loading loading-spinner"></span>
                    Downloading...
                  {:else}
                    🎲 Random Wallpaper
                  {/if}
                </button>
              </div>
            </div>
          </div>

          <!-- Current Wallpaper Preview -->
          {#if currentWallpaper}
            <div class="card bg-base-100 shadow-xl">
              <div class="card-body">
                <h3 class="card-title">🎯 Current Wallpaper</h3>
                
                <div class="flex flex-col lg:flex-row gap-6">
                  <div class="lg:w-2/3">
                    <figure class="aspect-video bg-base-200 rounded-lg overflow-hidden">
                      {#if imageCache.has(currentWallpaper.id)}
                        <img 
                          src={imageCache.get(currentWallpaper.id)} 
                          alt="Current wallpaper"
                          class="object-cover w-full h-full"
                          loading="lazy"
                          width="300"
                          height="200"
                        />
                      {:else}
                        <div class="flex items-center justify-center w-full h-full">
                          <div class="flex flex-col items-center gap-4">
                            <div class="text-6xl">🖼️</div>
                            <button 
                              class="btn btn-sm btn-ghost"
                              on:click={() => handleImageVisible(currentWallpaper)}
                            >
                              Load Preview
                            </button>
                          </div>
                        </div>
                      {/if}
                    </figure>
                  </div>
                  
                  <div class="lg:w-1/3 space-y-4">
                    <div class="stats stats-vertical shadow">
                      <div class="stat">
                        <div class="stat-title">File Size</div>
                        <div class="stat-value text-sm">{formatFileSize(currentWallpaper.file_size)}</div>
                      </div>
                      <div class="stat">
                        <div class="stat-title">Downloaded</div>
                        <div class="stat-value text-xs">{formatDate(currentWallpaper.download_date)}</div>
                      </div>
                    </div>
                    
                    <div class="flex flex-col gap-2">
                      <button 
                        class="btn btn-accent"
                        on:click={() => handleSet(currentWallpaper.filepath, currentWallpaper.filename)}
                      >
                        🎯 Set as Wallpaper
                      </button>
                      <button 
                        class="btn btn-ghost btn-sm"
                        on:click={handleOpenDirectory}
                      >
                        📁 Open Folder
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          {/if}

          <!-- Quick Actions -->
          <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div class="card bg-gradient-to-r from-blue-500/20 to-purple-500/20 shadow-xl">
              <div class="card-body text-center">
                <h3 class="card-title justify-center">🎨 High Quality</h3>
                <p class="text-sm">2K & 4K resolution wallpapers from premium sources</p>
              </div>
            </div>
            
            <div class="card bg-gradient-to-r from-green-500/20 to-blue-500/20 shadow-xl">
              <div class="card-body text-center">
                <h3 class="card-title justify-center">⚡ Auto-Set</h3>
                <p class="text-sm">Automatically applies wallpaper after download</p>
              </div>
            </div>
            
            <div class="card bg-gradient-to-r from-purple-500/20 to-pink-500/20 shadow-xl">
              <div class="card-body text-center">
                <h3 class="card-title justify-center">🔄 Auto-Change</h3>
                <p class="text-sm">Set automatic wallpaper rotation in settings</p>
              </div>
            </div>
          </div>
        </div>
      </div>

    <!-- Gallery Tab -->
    {:else if currentTab === 'gallery'}
      <div class="p-6">
        <div class="max-w-7xl mx-auto">
          
          <!-- Gallery Header -->
          <div class="flex justify-between items-center mb-6">
            <div>
              <h3 class="text-2xl font-bold">🖼️ Wallpaper Gallery</h3>
              <!-- <p class="text-base-content/70">Manage your downloaded wallpapers</p> -->
            </div>
            <button 
              class="btn btn-ghost gap-2"
              on:click={handleOpenDirectory}
            >
              📁 Open Folder
            </button>
          </div>

          {#if wallpapers.length === 0}
            <!-- Empty State -->
            <div class="card bg-base-100 shadow-xl">
              <div class="card-body text-center py-20">
                <div class="text-6xl mb-4">🖼️</div>
                <h3 class="text-xl font-bold mb-2">No wallpapers yet</h3>
                <p class="text-base-content/70 mb-6">Download your first wallpaper to get started</p>
                <button 
                  class="btn btn-primary"
                  on:click={() => currentTab = 'download'}
                >
                  🆕 Download Wallpaper
                </button>
              </div>
            </div>
          {:else}
            <!-- Wallpaper Grid -->
            <!-- <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 gap-6"> -->
          <div class="grid grid-cols-2 gap-2">
            {#each wallpapers as wallpaper}
                <div class="card bg-base-100 shadow-xl hover:shadow-2xl transition-all duration-300">
                  <figure class="aspect-video bg-base-200">
                    {#if imageCache.has(wallpaper.id)}
                      <img 
                        src={imageCache.get(wallpaper.id)} 
                        alt="Wallpaper preview"
                        class="object-cover w-full h-full"
                        loading="lazy"
                        width="150"
                        height="100"
                      />
                    {:else}
                      <div class="flex items-center justify-center w-full h-full">
                        <div class="flex flex-col items-center gap-2">
                          <span class="loading loading-spinner loading-sm"></span>
                          <button 
                            class="btn btn-xs btn-ghost"
                            on:click={() => handleImageVisible(wallpaper)}
                          >
                            Load
                          </button>
                        </div>
                      </div>
                    {/if}
                  </figure>
                  
                  <div class="card-body p-4">
                    <div class="text-xs text-base-content/70 mb-3">
                      <div>{formatDate(wallpaper.download_date)}</div>
                      <div class="badge badge-ghost badge-xs">{formatFileSize(wallpaper.file_size)}</div>
                    </div>
                    
                    <div class="card-actions justify-between">
                      <button 
                        class="btn btn-primary btn-sm flex-1"
                        on:click={() => handleSet(wallpaper.filepath, wallpaper.filename)}
                      >
                        🎯 Set
                      </button>
                      <button 
                        class="btn btn-error btn-sm btn-square"
                        on:click={() => handleDelete(wallpaper.id, wallpaper.filename)}
                        title="Delete wallpaper"
                      >
                        🗑️
                      </button>
                    </div>
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      </div>

    <!-- Settings Tab -->
    {:else if currentTab === 'settings' && settings}
      <div class="p-6">
        <div class="max-w-4xl mx-auto space-y-8">
          
          <!-- Settings Header -->
          <div class="text-center">
            <h2 class="text-2xl font-bold">⚙️ Settings</h2>
            <p class="text-base-content/70">Configure your wallpaper preferences</p>
          </div>

          <!-- Auto-Change Settings -->
          <div class="card bg-base-100 shadow-xl">
            <div class="card-body">
              <h3 class="card-title">🔄 Automatic Wallpaper Change</h3>
              
              <div class="form-control">
                <label class="label cursor-pointer">
                  <span class="label-text flex items-center gap-2">
                    <span class="text-lg">⏰</span>
                    Enable automatic wallpaper rotation
                  </span>
                  <input 
                    id="auto-change-toggle"
                    type="checkbox" 
                    class="toggle toggle-primary toggle-lg" 
                    bind:checked={settings.auto_change_enabled} 
                  />
                </label>
              </div>

              <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
                <div class="form-control">
                  <label class="label" for="interval-input">
                    <span class="label-text">🕐 Change interval (hours)</span>
                  </label>
                  <select 
                    id="interval-input"
                    class="select select-bordered" 
                    bind:value={settings.change_interval_hours}
                  >
                    <option value={1}>Every hour</option>
                    <option value={2}>Every 2 hours</option>
                    <option value={4}>Every 4 hours</option>
                    <option value={8}>Every 8 hours</option>
                    <option value={12}>Every 12 hours</option>
                    <option value={24}>Daily</option>
                  </select>
                </div>

                <div class="form-control">
                  <label class="label" for="max-wallpapers-input">
                    <span class="label-text">📂 Maximum wallpapers to keep</span>
                  </label>
                  <select 
                    id="max-wallpapers-input"
                    class="select select-bordered" 
                    bind:value={settings.max_wallpapers}
                  >
                    <option value={10}>10 wallpapers</option>
                    <option value={20}>20 wallpapers</option>
                    <option value={50}>50 wallpapers</option>
                    <option value={100}>100 wallpapers</option>
                  </select>
                </div>
              </div>
            </div>
          </div>

          <!-- Download Sources -->
          <div class="card bg-base-100 shadow-xl">
            <div class="card-body">
              <h3 class="card-title">🌐 Download Sources</h3>
              <p class="text-sm text-base-content/70 mb-4">
                Configure high-quality 2K/4K wallpaper sources (one URL per line):
              </p>
              
              <div class="form-control">
                <textarea 
                  id="sources-textarea"
                  class="textarea textarea-bordered h-40 font-mono text-sm" 
                  placeholder="https://source.unsplash.com/3840x2160/landscape&#10;https://source.unsplash.com/3840x2160/nature&#10;https://picsum.photos/3840/2160"
                  bind:value={sourcesText}
                ></textarea>
              </div>
              
              <div class="alert alert-info mt-4">
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" class="stroke-current shrink-0 w-6 h-6">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                </svg>
                <div class="text-sm">
                  <strong>🎨 Available sources:</strong><br>
                  • Unsplash 4K (3840x2160) - Nature, Landscape, Architecture<br>
                  • Unsplash 2K (2560x1440) - Cities, Space, Abstract<br>
                  • Picsum - Random high-quality photography<br>
                  • Custom URLs - Add your own image sources
                </div>
              </div>
            </div>
          </div>

          <!-- Save Button -->
          <div class="card bg-gradient-to-r from-primary/20 to-secondary/20 shadow-xl">
            <div class="card-body text-center">
              <button class="btn btn-primary btn-lg" on:click={handleSaveSettings}>
                💾 Save Settings
              </button>
            </div>
          </div>
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
