<script setup lang="ts">
import { search } from "~/lib/search";
import { logout } from "~/lib/auth";

const profile = await useProfile();
const showSearch = ref(false);
const showDropdown = ref(false);
const dropdownRef = ref<HTMLElement>();

const term = ref("");
async function doSearch() {
  if (await search(term.value)) {
    term.value = "";
    showSearch.value = false;
  }
}

function handleClick(e: PointerEvent) {
  const target = e.target as HTMLElement;
  if (!dropdownRef.value!.contains(target)) {
    showDropdown.value = false;
  }
}

onMounted(() => {
  document.addEventListener("click", handleClick);
});

onBeforeUnmount(() => {
  document.removeEventListener("click", handleClick);
});
</script>

<template>
  <nav
    class="flex items-center gap-2 border border-gray-300 dark:border-gray-700 rounded-lg px-4 py-3 mb-5"
  >
    <nuxt-link href="/" class="mr-1" aria-label="Home">
      <img
        class="rounded-lg"
        src="/icon-32.webp"
        height="32"
        width="32"
        alt=""
      />
    </nuxt-link>

    <nuxt-link class="nav-link" href="/"> Queue </nuxt-link>

    <nuxt-link class="nav-link" href="/audit-log"> Audit log </nuxt-link>

    <div class="ml-auto flex items-center gap-2">
      <shared-search @toggle-search="showSearch = !showSearch" />
      <div ref="dropdownRef" class="relative">
        <button type="button" @click="showDropdown = !showDropdown">
          <shared-avatar
            :did="profile.did"
            :has-avatar="Boolean(profile.avatar)"
            resize="72x72"
            :size="32"
          />
        </button>
        <div
          v-if="showDropdown"
          class="absolute right-0 w-[150px] bg-white dark:bg-slate-600 rounded-lg overflow-hidden top-full mt-1 border dark:border-gray-700"
          role="menu"
        >
          <nuxt-link
            href="/settings"
            class="w-full hover:bg-gray-100 hover:dark:bg-slate-700 px-2 py-1 cursor-pointer flex items-center gap-1 border-b dark:border-gray-700"
            @click="showDropdown = false"
          >
            <icon-settings /> Settings
          </nuxt-link>
          <nuxt-link
            class="w-full hover:bg-gray-100 hover:dark:bg-slate-700 px-2 py-1 cursor-pointer flex items-center gap-1"
            @click="logout"
          >
            <icon-logout /> Log out
          </nuxt-link>
        </div>
      </div>
    </div>
  </nav>
  <div
    v-if="showSearch"
    class="flex text-sm px-4 py-3 mb-5 border border-gray-300 dark:border-gray-700 rounded-lg"
  >
    <input
      v-model="term"
      class="py-1 px-2 w-full rounded-l-lg border border-gray-300 text-black"
      type="text"
      placeholder="User handle or did"
      @keydown="$event.key === 'Enter' ? doSearch() : null"
    />
    <button
      class="text-white hover:bg-blue-600 dark:hover:bg-blue-700 disabled:bg-blue-300 disabled:dark:bg-blue-500 rounded-r-lg px-1 py-1"
      @click="doSearch"
    >
      <icon-search />
    </button>
  </div>
</template>

<style scoped>
.nav-link {
  @apply px-2 py-1 text-sm rounded-lg;
}

.nav-link.router-link-active,
.nav-link:hover {
  @apply bg-slate-600 bg-opacity-50;
}
</style>
