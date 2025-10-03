<script setup lang="ts">
import { ProfileViewDetailed } from "@atproto/api/dist/client/types/app/bsky/actor/defs";
import { getProfile } from "~/lib/cached-bsky";

const props = defineProps<{ did: string; hideAvatar?: boolean }>();

const profile = ref<ProfileViewDetailed>();

profile.value = await getProfile(props.did);

watch(
  () => props.did,
  async () => {
    profile.value = await getProfile(props.did);
  }
);
</script>

<template>
  <nuxt-link
    class="flex items-center underline hover:no-underline"
    :href="`/users/${profile?.did || did}`"
  >
    <shared-avatar
      v-if="!hideAvatar"
      class="mr-1"
      :did="profile?.did"
      resize="20x20"
      :has-avatar="Boolean(profile?.avatar)"
      :size="20"
    />
    {{ profile?.handle || did }}
  </nuxt-link>
</template>
