<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { CircleAlert, KeyRound, LoaderCircle, MoreHorizontal, Pencil, Plus, Trash2, UserCheck, UserX } from '@lucide/vue'
import { ref } from 'vue'
import { toast } from 'vue-sonner'

import CreateUserDialog from '@/components/users/CreateUserDialog.vue'
import EditUserDialog from '@/components/users/EditUserDialog.vue'
import ResetUserPasswordDialog from '@/components/users/ResetUserPasswordDialog.vue'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from '@/components/ui/empty'
import { getAuthStatus, type AuthUser } from '@/lib/api/auth'
import { deleteUser, getUsers, updateUser } from '@/lib/api/users'

const queryClient = useQueryClient()
const createOpen = ref(false)
const editOpen = ref(false)
const resetOpen = ref(false)
const deleteOpen = ref(false)
const selectedUser = ref<AuthUser>()

const usersQuery = useQuery({ queryKey: ['users'], queryFn: ({ signal }) => getUsers(signal) })
const authQuery = useQuery({ queryKey: ['auth-status'], queryFn: ({ signal }) => getAuthStatus(signal) })

async function refreshUsers(): Promise<void> {
  await queryClient.invalidateQueries({ queryKey: ['users'] })
}

const activeMutation = useMutation({
  mutationFn: (user: AuthUser) => updateUser(user.id, { username: user.username, role: user.role, active: !user.active }),
  onSuccess: async (user) => { await refreshUsers(); toast.success(user.active ? 'Utilisateur activé' : 'Utilisateur désactivé') },
  onError: (error) => toast.error(error instanceof Error ? error.message : 'Impossible de modifier l’utilisateur'),
})
const deleteMutation = useMutation({
  mutationFn: () => {
    const user = selectedUser.value
    if (!user) throw new Error('Aucun utilisateur sélectionné')
    return deleteUser(user.id)
  },
  onSuccess: async () => { await refreshUsers(); deleteOpen.value = false; toast.success('Utilisateur supprimé') },
  onError: (error) => toast.error(error instanceof Error ? error.message : 'Impossible de supprimer l’utilisateur'),
})

function openEdit(user: AuthUser): void {
  selectedUser.value = user
  editOpen.value = true
}
function openReset(user: AuthUser): void {
  selectedUser.value = user
  resetOpen.value = true
}
function openDelete(user: AuthUser): void {
  selectedUser.value = user
  deleteOpen.value = true
}
function isCurrentUser(user: AuthUser): boolean {
  return authQuery.data.value?.user?.id === user.id
}

async function handleUserUpdated(): Promise<void> {
  await Promise.all([refreshUsers(), queryClient.invalidateQueries({ queryKey: ['auth-status'] })])
}
</script>

<template>
  <section class="min-h-[calc(100dvh-4rem)]">
    <div class="mx-auto max-w-5xl px-4 py-8 sm:px-5 sm:py-10 lg:px-10 lg:py-14">
      <div class="flex flex-wrap items-end justify-between gap-5">
        <div><h1 class="text-3xl font-bold tracking-tight sm:text-4xl">Utilisateurs</h1><p class="mt-2 text-sm text-muted-foreground">Gérez les personnes autorisées à accéder à Flex.</p></div>
        <Button @click="createOpen = true"><Plus />Ajouter</Button>
      </div>

      <div v-if="usersQuery.isPending.value" class="grid min-h-96 place-items-center"><LoaderCircle class="size-7 animate-spin text-primary" /></div>
      <Empty v-else-if="usersQuery.isError.value" class="mt-10 min-h-80 border border-red-400/15"><EmptyHeader><EmptyMedia variant="icon"><CircleAlert /></EmptyMedia><EmptyTitle>Impossible de charger les utilisateurs</EmptyTitle><EmptyDescription>Vérifiez votre session puis réessayez.</EmptyDescription></EmptyHeader><EmptyContent><Button variant="secondary" @click="usersQuery.refetch()">Réessayer</Button></EmptyContent></Empty>
      <div v-else class="mt-10 grid gap-3">
        <Card v-for="user in usersQuery.data.value" :key="user.id" class="gap-0 border-white/8 bg-card/60 py-0 shadow-none">
          <CardContent class="flex items-center gap-4 p-4 sm:p-5">
            <Avatar class="size-11"><AvatarFallback class="bg-primary/12 font-semibold text-primary">{{ user.username.slice(0, 1).toUpperCase() }}</AvatarFallback></Avatar>
            <div class="min-w-0 flex-1"><div class="flex flex-wrap items-center gap-2"><h2 class="truncate font-semibold">{{ user.username }}</h2><Badge v-if="isCurrentUser(user)" variant="outline">Vous</Badge></div><div class="mt-1 flex items-center gap-2 text-xs text-muted-foreground"><span>{{ user.role === 'admin' ? 'Administrateur' : 'Utilisateur' }}</span><span>·</span><span :class="user.active ? 'text-emerald-400' : 'text-amber-300'">{{ user.active ? 'Actif' : 'Désactivé' }}</span></div></div>
            <DropdownMenu>
              <DropdownMenuTrigger as-child><Button variant="ghost" size="icon" aria-label="Actions utilisateur"><MoreHorizontal /></Button></DropdownMenuTrigger>
              <DropdownMenuContent align="end" class="w-56">
                <DropdownMenuItem @select="openEdit(user)"><Pencil />Modifier</DropdownMenuItem>
                <DropdownMenuItem :disabled="isCurrentUser(user) || activeMutation.isPending.value" @select="activeMutation.mutate(user)"><UserX v-if="user.active" /><UserCheck v-else />{{ user.active ? 'Désactiver' : 'Activer' }}</DropdownMenuItem>
                <DropdownMenuItem @select="openReset(user)"><KeyRound />Réinitialiser le mot de passe</DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem variant="destructive" :disabled="isCurrentUser(user)" @select="openDelete(user)"><Trash2 />Supprimer</DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </CardContent>
        </Card>
      </div>
    </div>

    <CreateUserDialog v-model:open="createOpen" @created="refreshUsers" />
    <EditUserDialog v-model:open="editOpen" :user="selectedUser" :current-user="selectedUser ? isCurrentUser(selectedUser) : false" @updated="handleUserUpdated" />
    <ResetUserPasswordDialog v-model:open="resetOpen" :user="selectedUser" />

    <Dialog v-model:open="deleteOpen"><DialogContent class="border-white/10 bg-background sm:max-w-md"><DialogHeader><DialogTitle>Supprimer {{ selectedUser?.username }} ?</DialogTitle><DialogDescription>Le compte et toutes ses sessions seront définitivement supprimés.</DialogDescription></DialogHeader><DialogFooter><Button variant="ghost" @click="deleteOpen = false">Annuler</Button><Button variant="destructive" :disabled="deleteMutation.isPending.value" @click="deleteMutation.mutate()"><LoaderCircle v-if="deleteMutation.isPending.value" class="animate-spin" />Supprimer</Button></DialogFooter></DialogContent></Dialog>
  </section>
</template>
