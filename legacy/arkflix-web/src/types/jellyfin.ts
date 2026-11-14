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

export interface PersonInfo {
  Name: string
  Id: string
  Role?: string
  Type?: string
  PrimaryImageTag?: string
}

export interface MediaItem {
  Id: string
  Name: string
  OriginalTitle?: string
  Overview?: string
  Taglines?: string[]
  Type: 'Movie' | 'Series' | 'Episode' | 'Season' | 'CollectionFolder' | 'Folder'
  RunTimeTicks?: number
  ProductionYear?: number
  PremiereDate?: string
  EndDate?: string
  CommunityRating?: number
  CriticRating?: number
  OfficialRating?: string
  Genres?: string[]
  Studios?: { Id: string; Name: string }[]
  People?: PersonInfo[]
  SeriesName?: string
  SeriesId?: string
  SeasonName?: string
  SeasonId?: string
  IndexNumber?: number
  ParentIndexNumber?: number
  ChildCount?: number
  RecursiveItemCount?: number
  MediaStreams?: MediaStream[]
  Container?: string
  Path?: string
  UserData?: {
    PlaybackPositionTicks?: number
    PlayCount?: number
    IsFavorite?: boolean
    Played?: boolean
    PlayedPercentage?: number
    LastPlayedDate?: string
  }
  ImageTags?: {
    Primary?: string
    Backdrop?: string
    Logo?: string
    Thumb?: string
    Banner?: string
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
  Width?: number
  Height?: number
  AspectRatio?: string
  BitRate?: number
  ChannelLayout?: string
}
