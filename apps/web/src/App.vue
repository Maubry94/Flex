<script setup lang="ts">
import { Film, Heart, Menu, Search } from '@lucide/vue'
import { nextTick, ref } from 'vue'

import GlobalSearch from '@/components/search/GlobalSearch.vue'
import { Button } from '@/components/ui/button'
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle, SheetTrigger } from '@/components/ui/sheet'
import { Toaster } from '@/components/ui/sonner'

const isMobileMenuOpen = ref(false)
const isSearchOpen = ref(false)

async function openMobileSearch(): Promise<void> {
  isMobileMenuOpen.value = false
  await nextTick()
  isSearchOpen.value = true
}
</script>

<template>
  <div class="min-h-dvh bg-background text-foreground">
    <header class="fixed inset-x-0 top-0 z-50 border-b border-white/6 bg-background/80 backdrop-blur-xl">
      <div class="mx-auto flex h-16 max-w-[1600px] items-center gap-8 px-5 lg:px-10">
        <RouterLink class="flex items-center gap-2.5" to="/" aria-label="Accueil Flex">
          <span class="grid size-9 place-items-center rounded-xl bg-primary text-primary-foreground shadow-lg shadow-primary/25">
            <Film class="size-5" />
          </span>
          <span class="text-xl font-bold tracking-tight">Flex</span>
        </RouterLink>

        <nav class="hidden items-center gap-1 md:flex" aria-label="Navigation principale">
          <RouterLink class="rounded-full px-4 py-2 text-sm font-medium text-muted-foreground transition hover:text-foreground" active-class="bg-white/8 text-foreground" :to="{ name: 'home' }">Accueil</RouterLink>
          <RouterLink class="rounded-full px-4 py-2 text-sm font-medium text-muted-foreground transition hover:text-foreground" active-class="bg-white/8 text-foreground" :to="{ name: 'libraries' }">Bibliothèques</RouterLink>
          <RouterLink class="rounded-full px-4 py-2 text-sm font-medium text-muted-foreground transition hover:text-foreground" active-class="bg-white/8 text-foreground" :to="{ name: 'favorites' }">Favoris</RouterLink>
        </nav>

        <div class="ml-auto flex items-center gap-1">
          <button class="hidden size-10 place-items-center rounded-full text-muted-foreground transition hover:bg-white/8 hover:text-foreground md:grid" aria-label="Rechercher" @click="isSearchOpen = true">
            <Search class="size-5" />
          </button>
          <Sheet v-model:open="isMobileMenuOpen">
            <SheetTrigger as-child>
              <Button class="md:hidden" variant="ghost" size="icon" aria-label="Ouvrir la navigation">
                <Menu class="size-5" />
              </Button>
            </SheetTrigger>
            <SheetContent side="left" class="w-[min(86vw,22rem)] border-white/10 bg-background p-0">
              <SheetHeader class="border-b border-white/8 p-5 text-left">
                <SheetTitle class="flex items-center gap-2.5 text-lg">
                  <span class="grid size-9 place-items-center rounded-xl bg-primary text-primary-foreground"><Film class="size-5" /></span>
                  Flex
                </SheetTitle>
                <SheetDescription class="sr-only">Navigation principale de Flex</SheetDescription>
              </SheetHeader>
              <nav class="flex flex-col gap-1 p-4" aria-label="Navigation mobile">
                <button type="button" class="flex items-center gap-3 rounded-xl px-4 py-3 text-left text-sm font-medium text-muted-foreground transition hover:bg-white/5 hover:text-foreground" @click="openMobileSearch"><Search class="size-4" />Rechercher</button>
                <RouterLink class="rounded-xl px-4 py-3 text-sm font-medium text-muted-foreground transition hover:bg-white/5 hover:text-foreground" active-class="bg-white/8 text-foreground" :to="{ name: 'home' }" @click="isMobileMenuOpen = false">Accueil</RouterLink>
                <RouterLink class="rounded-xl px-4 py-3 text-sm font-medium text-muted-foreground transition hover:bg-white/5 hover:text-foreground" active-class="bg-white/8 text-foreground" :to="{ name: 'libraries' }" @click="isMobileMenuOpen = false">Bibliothèques</RouterLink>
                <RouterLink class="flex items-center gap-3 rounded-xl px-4 py-3 text-sm font-medium text-muted-foreground transition hover:bg-white/5 hover:text-foreground" active-class="bg-white/8 text-foreground" :to="{ name: 'favorites' }" @click="isMobileMenuOpen = false"><Heart class="size-4" />Favoris</RouterLink>
              </nav>
            </SheetContent>
          </Sheet>
        </div>
      </div>
    </header>

    <main class="pt-16">
      <RouterView />
    </main>
    <GlobalSearch v-model:open="isSearchOpen" />
    <Toaster position="bottom-right" rich-colors close-button />
  </div>
</template>
