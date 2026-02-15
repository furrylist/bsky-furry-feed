<script setup lang="ts">
import {
  showQueueActionConfirmation,
  blurNsfwPostMedia,
  useKeyboardShurtcuts,
} from "~/lib/settings";
import { logout } from "~/lib/auth";
</script>

<template>
  <div>
    <h1 class="text-xl font-bold">My settings</h1>
    <p class="text-muted mb-4">Settings aren’t synced across your devices.</p>

    <div class="flex gap-2 mb-4">
      <input
        id="show-queue-action-confirmation"
        v-model="showQueueActionConfirmation"
        type="checkbox"
      />
      <label for="show-queue-action-confirmation"
        >Show confirmation popup when approving and rejecting in the
        queue.</label
      >
    </div>
    <div class="flex gap-2 mb-4">
      <input
        id="blur-nsfw-post-media"
        v-model="blurNsfwPostMedia"
        type="checkbox"
      />
      <label for="blur-nsfw-post-media"
        >Blur NSFW post media (images/video).</label
      >
    </div>
    <div class="flex gap-2 mb-4">
      <input
        id="use-keyboard-shortcuts"
        v-model="useKeyboardShurtcuts"
        type="checkbox"
      />
      <label for="buse-keyboard-shortcuts"
        >Use keyboard shortcuts: <kbd>A</kbd> for approve, <kbd>R</kbd> for
        reject, and <kbd>H</kbd> for hold back</label
      >
    </div>

    <button class="text-white bg-gray-700 px-2 py-1 rounded-lg" @click="logout">
      Logout
    </button>
  </div>
</template>
