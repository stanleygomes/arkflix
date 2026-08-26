export interface JellyfinUser {
  Id: string
  Name: string
  ServerId?: string
  HasPassword?: boolean
  PrimaryImageTag?: string
}

export interface AuthenticationResult {
  User: JellyfinUser
  AccessToken: string
  ServerId: string
}

export interface MediaItem {
  Id: string
  Name: string
  OriginalTitle?: string
  Overview?: string
  Type: 'Movie' | 'Series' | 'Episode' | 'Season' | 'CollectionFolder' | 'Folder'
  RunTimeTicks?: number
  ProductionYear?: number
  CommunityRating?: number
  OfficialRating?: string
  Genres?: string[]
  SeriesName?: string
  SeriesId?: string
  SeasonName?: string
  SeasonId?: string
  IndexNumber?: number
  ParentIndexNumber?: number
  UserData?: {
    PlaybackPositionTicks?: number
    PlayCount?: number
    IsFavorite?: boolean
    Played?: boolean
    PlayedPercentage?: number
  }
  ImageTags?: {
    Primary?: string
    Backdrop?: string
    Logo?: string
    Thumb?: string
  }
  BackdropImageTags?: string[]
}

export interface MediaLibrary {
  Id: string
  Name: string
  CollectionType?: 'movies' | 'tvshows' | 'music' | 'homevideos' | string
}

export interface PlaybackInfo {
  MediaSources: MediaSourceInfo[]
  PlaySessionId: string
}

export interface MediaSourceInfo {
  Id: string
  Path?: string
  Protocol?: string
  Container?: string
  MediaStreams: MediaStream[]
  SupportsDirectPlay: boolean
  SupportsDirectStream: boolean
  SupportsTranscoding: boolean
}

export interface MediaStream {
  Type: 'Audio' | 'Video' | 'Subtitle'
  Index: number
  Codec: string
  Language?: string
  DisplayTitle?: string
  IsDefault: boolean
  IsForced: boolean
  DeliveryUrl?: string
}
