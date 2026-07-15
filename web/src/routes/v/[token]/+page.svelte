<script lang="ts">import {page} from '$app/stores';import Board from '$lib/Board.svelte';$:token=$page.params.token??'';</script><Board {token} mode="view"/>
