<script setup lang="ts">
import * as auth from "~/lib/auth";

const identifier = ref<string>("");
const password = ref<string>("");
const error = ref<any>();

async function login() {
  error.value = null;

  const isSignedIn = await auth
    .login(identifier.value?.replace(/^@/, ""), password.value)
    .catch((error) => ({ error }));

  if (isSignedIn.error) {
    error.value = {
      message: isSignedIn.error,
    };
  }
}
</script>

<template>
  <div class="flex items-center justify-center fixed w-full h-full">
    <div class="mx-auto flex flex-col gap-8">
      <div class="flex justify-center items-center gap-3">
        <img class="rounded-lg" src="/favicon.ico" height="48" width="48" />
        <h1 class="text-2xl font-bold">Furrylist Admin</h1>
      </div>
      <div
        class="bg-gray-50 border border-gray-400 dark:border-gray-700 dark:bg-gray-800 py-4 px-5 rounded-lg w-[400px] max-w-[80vw]"
      >
        <div class="flex flex-col mb-4">
          <label class="mb-1" for="name">Handle</label>
          <input
            id="name"
            v-model="identifier"
            class="bg-white dark:bg-gray-900 rounded border border-gray-400 dark:border-gray-700 px-2 py-1"
            type="text"
          />
        </div>

        <div class="flex flex-col mb-4">
          <label class="mb-1" for="password">App password</label>
          <input
            id="password"
            v-model="password"
            class="bg-white dark:bg-gray-900 rounded border border-gray-400 dark:border-gray-700 px-2 py-1"
            type="password"
          />
        </div>

        <div class="flex">
          <label v-if="error" class="mr-auto px-1 py-2 text-red-600">
            {{ error.message }}
          </label>
          <button
            class="ml-auto px-3 py-1.5 rounded-lg bg-blue-600"
            @click="login"
          >
            Login
          </button>
        </div>
      </div>
      <nuxt-link
        href="https://furryli.st"
        class="text-muted underline hover:no-underline text-center"
      >
        &larr; Back to home
      </nuxt-link>
    </div>
  </div>
</template>
