// Transporte da API. No desktop, o Wails injeta window.go.core.Service (cada
// método do Go vira uma Promise) e window.runtime (eventos). No navegador, os
// mesmos métodos vão por POST /api/call/<Método> e os eventos por SSE.
// Trocar pra um servidor remoto no futuro = mudar só a URL base daqui.
import type {
  Boot,
  Game,
  GameConfigInfo,
  GameList,
  ListRule,
  WikiResult,
  GameSettings,
  GameSizeInfo,
  BTInfo,
  CheevosInfo,
  BiosInfo,
  EmulatorsInfo,
  GameCheevos,
  UpdateInfo,
  GuideInfo,
  MediaFile,
  KidsTimeInfo,
  ImportItem,
  ImportResult,
  InputResult,
  Listing,
  Mapping,
  MetaEdit,
  MetaInfo,
  MoveTarget,
  Pad,
  Profile,
  ProfileState,
  RemoteInfo,
  StorageInfo,
  ScriptsInfo,
  Settings,
  DownloadOptions,
  TaskEvent,
  Stats,
  Accounts,
  WatchInfo,
  WatchHome,
  WatchPage,
  WatchDetails,
  WatchItem,
  WatchStream,
  ArkInfo,
  ArkKind,
  ArkList,
  ArkPlatform,
  ArkStatus,
  ArkUploadKind,
  ArkUser,
  MusicInfo,
  MusicServer,
  MusicHome,
  MusicIndex,
  MusicAlbum,
  MusicTrack,
  MusicArtist,
  MusicPlaylist,
  MusicFound,
  MusicLocalSet,
  MusicStream,
  MusicAdminInfo,
} from './types'

type Service = Record<string, (...args: unknown[]) => Promise<unknown>>

interface WailsWindow {
  go?: { core?: { Service?: Service } }
  runtime?: {
    EventsOn: (name: string, cb: (data: unknown) => void) => (() => void) | void
    EventsOff: (name: string) => void
  }
}

const w = window as unknown as WailsWindow

export const isDesktop = (): boolean => typeof w.runtime?.EventsOn === 'function'

async function binding(): Promise<Service> {
  for (let i = 0; i < 100; i++) {
    const svc = w.go?.core?.Service
    if (svc) return svc
    await new Promise((r) => setTimeout(r, 50))
  }
  throw new Error('ligação com o Go não ficou pronta')
}

export async function call<T>(method: string, ...args: unknown[]): Promise<T> {
  if (isDesktop()) {
    const svc = await binding()
    const fn = svc[method]
    if (!fn) throw new Error(`método ${method} não existe`)
    try {
      return (await fn(...args)) as T
    } catch (err) {
      throw new Error(typeof err === 'string' ? err : String((err as Error)?.message ?? err))
    }
  }
  const res = await fetch(`/api/call/${method}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-Arkflix': '1' },
    body: JSON.stringify(args),
  })
  const body = (await res.json().catch(() => ({}))) as { result?: T; error?: string }
  if (!res.ok) throw new Error(body.error ?? `HTTP ${res.status}`)
  return body.result as T
}

type Listener = (data: unknown) => void
const listeners = new Map<string, Set<Listener>>()
let source: EventSource | null = null

function ensureSource() {
  if (source || isDesktop()) return
  source = new EventSource('/api/events')
  source.onmessage = (msg) => {
    const { name, data } = JSON.parse(msg.data) as { name: string; data: unknown }
    listeners.get(name)?.forEach((fn) => fn(data))
  }
}

/** Assina um evento do Go ("pad", "pads", "game", "library"). Devolve o cancelamento. */
export function on<T>(name: string, fn: (data: T) => void): () => void {
  const cb = fn as Listener
  if (isDesktop()) {
    const off = w.runtime!.EventsOn(name, cb)
    return typeof off === 'function' ? off : () => w.runtime!.EventsOff(name)
  }
  ensureSource()
  if (!listeners.has(name)) listeners.set(name, new Set())
  listeners.get(name)!.add(cb)
  return () => listeners.get(name)?.delete(cb)
}

export const api = {
  bootstrap: () => call<Boot>('Bootstrap'),
  profiles: () => call<ProfileState>('Profiles'),
  selectProfile: (id: string, pin: string) => call<Boot>('SelectProfile', id, pin),
  createProfile: (name: string, avatar: string, color: string) => call<Profile>('CreateProfile', name, avatar, color),
  updateProfile: (id: string, name: string, avatar: string, color: string) => call<Profile>('UpdateProfile', id, name, avatar, color),
  setProfilePIN: (id: string, current: string, next: string) => call<Profile>('SetProfilePIN', id, current, next),
  checkProfilePIN: (id: string, pin: string) => call<void>('CheckProfilePIN', id, pin),
  deleteProfile: (id: string, pin: string) => call<void>('DeleteProfile', id, pin),
  setProfileKids: (id: string, kids: boolean) => call<Profile>('SetProfileKids', id, kids),
  setProfileLimits: (id: string, minutes: number, bedtime: string) => call<Profile>('SetProfileLimits', id, minutes, bedtime),
  kidsTime: () => call<KidsTimeInfo>('KidsTime'),
  remoteStatus: () => call<RemoteInfo>('RemoteStatus'),
  storage: () => call<StorageInfo>('Storage'),
  movePlan: (system: string, from: string) => call<MoveTarget[]>('MovePlan', system, from),
  moveGames: (system: string, from: string, to: string) => call<void>('MoveGames', system, from, to),
  connectRemote: (host: string, user: string, pass: string) => call<RemoteInfo>('ConnectRemote', host, user, pass),
  disconnectRemote: () => call<void>('DisconnectRemote'),
  refreshRemote: () => call<void>('RefreshRemote'),
  downloadGame: (id: string) => call<void>('DownloadGame', id),
  downloads: () => call<TaskEvent[]>('Downloads'),
  pauseDownload: (id: string) => call<void>('PauseDownload', id),
  resumeDownload: (id: string) => call<void>('ResumeDownload', id),
  downloadNow: (id: string) => call<void>('DownloadNow', id),
  setDownloadOptions: (d: DownloadOptions) => call<Settings>('SetDownloadOptions', d),
  ready: () => call<void>('Ready'),
  refresh: () => call<Game[]>('Refresh'),
  setFavorite: (id: string, fav: boolean) => call<Game>('SetFavorite', id, fav),
  setHidden: (id: string, hidden: boolean) => call<Game>('SetHidden', id, hidden),
  gameSize: (id: string) => call<GameSizeInfo>('GameSize', id),
  deleteGame: (id: string) => call<void>('DeleteGame', id),
  gameConfig: (id: string) => call<GameConfigInfo>('GameConfig', id),
  setGameConfig: (id: string, c: GameSettings) => call<GameConfigInfo>('SetGameConfig', id, c),
  play: (id: string, state = '') => call<void>('Play', id, state),
  saves: (id: string) => call<Listing>('Saves', id),
  deleteState: (id: string, state: string) => call<Listing>('DeleteState', id, state),
  openManual: (id: string) => call<void>('OpenManual', id),
  analyzeImport: (paths: string[]) => call<ImportItem[]>('AnalyzeImport', paths),
  importItems: (items: ImportItem[]) => call<ImportResult[]>('Import', items),
  pickFiles: () => call<string[] | null>('PickFiles'),
  getSettings: () => call<Settings>('GetSettings'),
  saveSettings: (s: Settings) => call<Settings>('SaveSettings', s),
  resetSettings: () => call<Settings>('ResetSettings'),
  getInput: () => call<Mapping>('GetInput'),
  saveInput: (m: Mapping) => call<InputResult>('SaveInput', m),
  resetInput: () => call<InputResult>('ResetInput'),
  pads: () => call<Pad[]>('Pads'),
  scripts: () => call<ScriptsInfo>('Scripts'),
  updateCovers: (all = false) => call<void>('UpdateCovers', all),
  lists: () => call<GameList[]>('Lists'),
  createList: (name: string) => call<GameList>('CreateList', name),
  createSmartList: (name: string, rule: ListRule) => call<GameList>('CreateSmartList', name, rule),
  renameList: (id: string, name: string) => call<GameList>('RenameList', id, name),
  deleteList: (id: string) => call<void>('DeleteList', id),
  setInList: (id: string, game: string, inList: boolean) => call<GameList>('SetInList', id, game, inList),
  metaDetails: (id: string) => call<MetaInfo>('MetaDetails', id),
  fetchMeta: (id: string) => call<WikiResult>('FetchMeta', id),
  missingMeta: () => call<number>('MissingMeta'),
  completeMeta: () => call<void>('CompleteMeta'),
  stats: (period: string) => call<Stats>('Stats', period),
  saveMeta: (id: string, edit: MetaEdit) => call<Game>('SaveMeta', id, edit),
  resetMeta: (id: string) => call<Game>('ResetMeta', id),
  toggleFullscreen: () => call<void>('ToggleFullscreen'),
  quit: () => call<void>('Quit'),
  minimize: () => call<void>('Minimize'),
  isFullscreen: () => call<boolean>('IsFullscreen'),
  windowState: () => call<{ fullscreen: boolean; maximised: boolean }>('Window'),
  toggleMaximise: () => call<void>('ToggleMaximise'),
  cancelTask: (id: string) => call<void>('CancelTask', id),
  emulatorStatus: () => call<EmulatorsInfo>('EmulatorStatus'),
  installEmulators: (which: string[] = []) => call<void>('InstallEmulators', which),
  biosStatus: () => call<BiosInfo[]>('BiosStatus'),
  openBiosFolder: (system: string) => call<void>('OpenBiosFolder', system),
  btStatus: () => call<BTInfo>('BluetoothStatus'),
  btScan: (seconds = 20) => call<void>('BluetoothScan', seconds),
  btPower: (on: boolean) => call<void>('BluetoothPower', on),
  btPair: (addr: string) => call<void>('BluetoothPair', addr),
  btForget: (addr: string) => call<void>('BluetoothForget', addr),
  btConnect: (addr: string) => call<void>('BluetoothConnect', addr),
  btDisconnect: (addr: string) => call<void>('BluetoothDisconnect', addr),
  btSettings: () => call<void>('OpenBluetoothSettings'),
  cheevosStatus: () => call<CheevosInfo>('CheevosStatus'),
  cheevosLogin: (user: string, pass: string) => call<CheevosInfo>('CheevosLogin', user, pass),
  cheevosLogout: () => call<void>('CheevosLogout'),
  gameCheevos: (id: string) => call<GameCheevos>('GameCheevos', id),
  updateStatus: () => call<UpdateInfo>('UpdateStatus'),
  checkUpdate: () => call<UpdateInfo>('CheckUpdate'),
  downloadUpdate: () => call<void>('DownloadUpdate'),
  updateNow: () => call<void>('UpdateNow'),
  guide: () => call<GuideInfo>('Guide'),
  showGuide: () => call<void>('ShowGuide'),
  resumeGame: () => call<void>('ResumeGame'),
  gameCommand: (cmd: 'save' | 'load' | 'quit' | 'fps' | 'screenshot' | 'record') => call<void>('GameCommand', cmd),
  switchGame: (id: string) => call<void>('SwitchGame', id),
  gameMedia: (id: string) => call<MediaFile[]>('GameMedia', id),
  gameNote: (id: string) => call<string>('GameNote', id),
  setGameNote: (id: string, text: string) => call<void>('SetGameNote', id, text),
  // filmes e séries (Jellyfin)
  watchStatus: () => call<WatchInfo>('WatchStatus'),
  profileAccounts: (profile: string) => call<Accounts>('ProfileAccounts', profile),
  linkRomM: (profile: string) => call<Accounts>('LinkRomM', profile),
  connectWatch: (profile: string, host: string, user: string, pass: string) => call<WatchInfo>('ConnectWatch', profile, host, user, pass),
  disconnectWatch: (profile: string) => call<void>('DisconnectWatch', profile),
  watchHome: () => call<WatchHome>('WatchHome'),
  watchList: (kind: string, sort: string, genre = '', start = 0, limit = 60) => call<WatchPage>('WatchList', kind, sort, genre, start, limit),
  watchSearch: (text: string) => call<WatchItem[]>('WatchSearch', text),
  watchDetails: (id: string) => call<WatchDetails>('WatchDetails', id),
  watchEpisodes: (series: string, season: string) => call<WatchItem[]>('WatchEpisodes', series, season),
  watchPlay: (id: string, audio = -1, start = -1) => call<WatchStream>('WatchPlay', id, audio, start),
  watchReport: (event: 'start' | 'progress' | 'stop', id: string, session: string, source: string, position: number, paused: boolean) =>
    call<void>('WatchReport', event, id, session, source, position, paused),
  watchFavorite: (id: string, fav: boolean) => call<WatchItem>('WatchFavorite', id, fav),
  watchPlayed: (id: string, played: boolean) => call<WatchItem>('WatchPlayed', id, played),
  // arkflix-server: administração e listas (com o token que o app já tem)
  arkServer: () => call<ArkInfo>('ArkServer'),
  arkStatus: () => call<ArkStatus>('ArkStatus'),
  arkScan: () => call<void>('ArkScan'),
  arkUsers: () => call<ArkUser[]>('ArkUsers'),
  arkCreateUser: (name: string, password: string, admin: boolean) => call<ArkUser>('ArkCreateUser', name, password, admin),
  arkSetPassword: (id: string, password: string) => call<ArkUser>('ArkSetPassword', id, password),
  arkSetAdmin: (id: string, admin: boolean) => call<ArkUser>('ArkSetAdmin', id, admin),
  arkDeleteUser: (id: string) => call<void>('ArkDeleteUser', id),
  arkPickFiles: () => call<string[] | null>('ArkPickFiles'),
  arkPlatforms: () => call<ArkPlatform[]>('ArkPlatforms'),
  arkUpload: (kind: ArkUploadKind, fields: Record<string, string>, paths: string[]) => call<void>('ArkUpload', kind, fields, paths),
  arkHide: (kind: ArkKind, id: string) => call<void>('ArkHide', kind, id),
  arkLists: () => call<ArkList[]>('ArkLists'),
  arkList: (id: string) => call<ArkList>('ArkList', id),
  arkCreateList: (name: string) => call<ArkList>('ArkCreateList', name),
  arkRenameList: (id: string, name: string) => call<ArkList>('ArkRenameList', id, name),
  arkDeleteList: (id: string) => call<void>('ArkDeleteList', id),
  arkListsFor: (kind: ArkKind, id: string) => call<ArkList[]>('ArkListsFor', kind, id),
  arkSetInList: (list: string, kind: ArkKind, id: string, inList: boolean) => call<void>('ArkSetInList', list, kind, id, inList),
  // músicas (Subsonic/Navidrome): conta por perfil, áudio e capas pelo proxy
  musicStatus: () => call<MusicInfo>('MusicStatus'),
  connectMusic: (profile: string, host: string, user: string, pass: string, key = '') => call<MusicInfo>('ConnectMusic', profile, host, user, pass, key),
  disconnectMusic: (profile: string) => call<void>('DisconnectMusic', profile),
  musicServer: () => call<MusicServer>('MusicServerInfo'),
  musicHome: () => call<MusicHome>('MusicHome'),
  musicArtists: () => call<MusicIndex[]>('MusicArtists'),
  musicAlbums: (sort: string, start = 0, limit = 60) => call<{ albums: MusicAlbum[]; more: boolean }>('MusicAlbums', sort, start, limit),
  musicArtist: (id: string) => call<{ artist: MusicArtist; albums: MusicAlbum[]; top: MusicTrack[] }>('MusicArtist', id),
  musicAlbum: (id: string) => call<{ album: MusicAlbum; tracks: MusicTrack[]; offline: boolean }>('MusicAlbum', id),
  musicSearch: (text: string) => call<MusicFound>('MusicSearch', text),
  musicStarred: () => call<MusicFound>('MusicStarred'),
  musicStar: (kind: 'track' | 'album' | 'artist', id: string, on: boolean) => call<void>('MusicStar', kind, id, on),
  musicPlay: (id: string) => call<MusicStream>('MusicPlay', id),
  musicScrobble: (id: string, submission: boolean) => call<void>('MusicScrobble', id, submission),
  musicSaveQueue: (ids: string[], current: string, position: number) => call<void>('MusicSaveQueue', ids, current, position),
  musicResume: () => call<{ tracks: MusicTrack[]; current: number; position: number }>('MusicResume'),
  musicPlaylists: () => call<MusicPlaylist[]>('MusicPlaylists'),
  musicPlaylist: (id: string) => call<MusicPlaylist>('MusicPlaylist', id),
  musicCreatePlaylist: (name: string, tracks: string[] = []) => call<MusicPlaylist>('MusicCreatePlaylist', name, tracks),
  musicRenamePlaylist: (id: string, name: string) => call<void>('MusicRenamePlaylist', id, name),
  musicDeletePlaylist: (id: string) => call<void>('MusicDeletePlaylist', id),
  musicAddToPlaylist: (id: string, tracks: string[]) => call<void>('MusicAddToPlaylist', id, tracks),
  musicAddAlbumToPlaylist: (id: string, album: string) => call<void>('MusicAddAlbumToPlaylist', id, album),
  musicRemoveFromPlaylist: (id: string, positions: number[]) => call<void>('MusicRemoveFromPlaylist', id, positions),
  musicReorderPlaylist: (id: string, tracks: string[]) => call<void>('MusicReorderPlaylist', id, tracks),
  musicAdmin: () => call<MusicAdminInfo>('MusicAdmin'),
  musicScan: () => call<void>('MusicScan'),
  musicDownload: (kind: 'album' | 'playlist', id: string) => call<void>('MusicDownload', kind, id),
  musicRemoveLocal: (kind: 'album' | 'playlist', id: string) => call<void>('MusicRemoveLocal', kind, id),
  musicLocal: () => call<MusicLocalSet[]>('MusicLocal'),
}
