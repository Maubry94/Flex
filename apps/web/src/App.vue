<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { CircleUserRound, Film, Heart, House, Layers3, Library, LoaderCircle, LogOut, Menu, Search, ShieldCheck, UserRound, Users } from '@lucide/vue'
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'

import GlobalSearch from '@/components/search/GlobalSearch.vue'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { DropdownMenu, DropdownMenuContent, DropdownMenuGroup, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle, SheetTrigger } from '@/components/ui/sheet'
import { Toaster } from '@/components/ui/sonner'
import { getAuthStatus, logout } from '@/lib/api/auth'
import { authenticationRequiredEvent } from '@/lib/api/client'
import LoginView from '@/views/LoginView.vue'
import SetupView from '@/views/SetupView.vue'

const isMobileMenuOpen = ref(false)
const isSearchOpen = ref(false)
const queryClient = useQueryClient()
const authQuery = useQuery({ queryKey: ['auth-status'], queryFn: ({ signal }) => getAuthStatus(signal), retry: false })
const avatarInitial = computed(() => authQuery.data.value?.user?.username.trim().slice(0, 1).toUpperCase() || '?')
const logoutMutation = useMutation({
  mutationFn: logout,
  onSuccess: async () => {
    queryClient.clear()
    await authQuery.refetch()
  },
})

async function refreshAuthentication(): Promise<void> {
  await authQuery.refetch()
}

function handleAuthenticationRequired(): void {
  queryClient.clear()
  void authQuery.refetch()
}

onMounted(() => { window.addEventListener(authenticationRequiredEvent, handleAuthenticationRequired) })
onBeforeUnmount(() => { window.removeEventListener(authenticationRequiredEvent, handleAuthenticationRequired) })

async function openMobileSearch(): Promise<void> {
  isMobileMenuOpen.value = false
  await nextTick()
  isSearchOpen.value = true
}
</script>

<template>
  <div v-if="authQuery.isPending.value" class="grid min-h-dvh place-items-center bg-background text-foreground"><LoaderCircle class="size-7 animate-spin text-primary" /></div>
  <div v-else-if="authQuery.isError.value" class="grid min-h-dvh place-items-center bg-background px-4 text-foreground"><div class="text-center"><p class="text-sm text-muted-foreground">Impossible de contacter le serveur Flex.</p><Button class="mt-4" variant="secondary" @click="authQuery.refetch()">Réessayer</Button></div></div>
  <SetupView v-else-if="!authQuery.data.value?.configured" @authenticated="refreshAuthentication" />
  <LoginView v-else-if="!authQuery.data.value.authenticated" @authenticated="refreshAuthentication" />
  <div v-else class="min-h-dvh bg-background text-foreground">
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
          <RouterLink class="rounded-full px-4 py-2 text-sm font-medium text-muted-foreground transition hover:text-foreground" active-class="bg-white/8 text-foreground" :to="{ name: 'collections' }">Collections</RouterLink>
        </nav>

        <div class="ml-auto flex items-center gap-1">
          <button class="hidden size-10 place-items-center rounded-full text-muted-foreground transition hover:bg-white/8 hover:text-foreground md:grid" aria-label="Rechercher" @click="isSearchOpen = true">
            <Search class="size-5" />
          </button>
          <DropdownMenu>
            <DropdownMenuTrigger as-child>
              <button type="button" class="hidden rounded-full transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary md:inline-flex" aria-label="Ouvrir le menu du profil">
                <Avatar class="size-9 ring-1 ring-primary/25">
                  <AvatarFallback class="bg-primary/15 text-sm font-semibold text-primary transition hover:bg-primary/25">{{ avatarInitial }}</AvatarFallback>
                </Avatar>
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" class="w-56">
              <DropdownMenuLabel><p class="truncate text-sm">{{ authQuery.data.value.user?.username }}</p><p class="mt-0.5 text-xs font-normal text-muted-foreground">{{ authQuery.data.value.user?.role === 'admin' ? 'Administrateur' : 'Utilisateur' }}</p></DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuGroup>
                <DropdownMenuLabel class="text-xs text-muted-foreground">Compte</DropdownMenuLabel>
                <DropdownMenuItem as-child><RouterLink :to="{ name: 'account' }"><UserRound />Mon compte</RouterLink></DropdownMenuItem>
              </DropdownMenuGroup>
              <template v-if="authQuery.data.value.user?.role === 'admin'">
                <DropdownMenuSeparator />
                <DropdownMenuGroup>
                  <DropdownMenuLabel class="flex items-center gap-2 text-xs text-muted-foreground"><ShieldCheck class="size-3.5" />Administration</DropdownMenuLabel>
                  <DropdownMenuItem as-child><RouterLink :to="{ name: 'users' }"><Users />Utilisateurs</RouterLink></DropdownMenuItem>
                </DropdownMenuGroup>
              </template>
              <DropdownMenuSeparator />
              <DropdownMenuItem variant="destructive" :disabled="logoutMutation.isPending.value" @select="logoutMutation.mutate()"><LoaderCircle v-if="logoutMutation.isPending.value" class="animate-spin" /><LogOut v-else />Déconnexion</DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
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
                <RouterLink class="flex items-center gap-3 rounded-xl px-4 py-3 text-sm font-medium text-muted-foreground transition hover:bg-white/5 hover:text-foreground" active-class="bg-white/8 text-foreground" :to="{ name: 'home' }" @click="isMobileMenuOpen = false"><House class="size-4" />Accueil</RouterLink>
                <RouterLink class="flex items-center gap-3 rounded-xl px-4 py-3 text-sm font-medium text-muted-foreground transition hover:bg-white/5 hover:text-foreground" active-class="bg-white/8 text-foreground" :to="{ name: 'libraries' }" @click="isMobileMenuOpen = false"><Library class="size-4" />Bibliothèques</RouterLink>
                <RouterLink class="flex items-center gap-3 rounded-xl px-4 py-3 text-sm font-medium text-muted-foreground transition hover:bg-white/5 hover:text-foreground" active-class="bg-white/8 text-foreground" :to="{ name: 'favorites' }" @click="isMobileMenuOpen = false"><Heart class="size-4" />Favoris</RouterLink>
                <RouterLink class="flex items-center gap-3 rounded-xl px-4 py-3 text-sm font-medium text-muted-foreground transition hover:bg-white/5 hover:text-foreground" active-class="bg-white/8 text-foreground" :to="{ name: 'collections' }" @click="isMobileMenuOpen = false"><Layers3 class="size-4" />Collections</RouterLink>
                <div class="my-2 border-t border-white/8" />
                <template v-if="authQuery.data.value.user?.role === 'admin'">
                  <p class="flex items-center gap-2 px-4 py-2 text-xs font-medium text-muted-foreground"><ShieldCheck class="size-3.5" />Administration</p>
                  <RouterLink class="flex items-center gap-3 rounded-xl px-4 py-3 text-sm font-medium text-muted-foreground transition hover:bg-white/5 hover:text-foreground" active-class="bg-white/8 text-foreground" :to="{ name: 'users' }" @click="isMobileMenuOpen = false"><Users class="size-4" />Utilisateurs</RouterLink>
                  <div class="my-2 border-t border-white/8" />
                </template>
                <RouterLink class="flex items-center gap-3 rounded-xl px-4 py-3 text-sm font-medium text-muted-foreground transition hover:bg-white/5 hover:text-foreground" active-class="bg-white/8 text-foreground" :to="{ name: 'account' }" @click="isMobileMenuOpen = false"><CircleUserRound class="size-4" />Mon compte</RouterLink>
                <button type="button" class="flex items-center gap-3 rounded-xl px-4 py-3 text-left text-sm font-medium text-muted-foreground transition hover:bg-white/5 hover:text-foreground" :disabled="logoutMutation.isPending.value" @click="isMobileMenuOpen = false; logoutMutation.mutate()"><LogOut class="size-4" />Déconnexion</button>
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
