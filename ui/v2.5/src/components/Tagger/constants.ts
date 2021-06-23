export const LOCAL_FORAGE_KEY = "tagger";
export const DEFAULT_BLACKLIST = [
  "\\sXXX\\s",
  "1080p",
  "720p",
  "2160p",
  "KTR",
  "RARBG",
  "\\scom\\s",
  "\\[",
  "\\]",
];
export const DEFAULT_EXCLUDED_PERFORMER_FIELDS = ["name"];
export const DEFAULT_EXCLUDED_SCENE_FIELDS = [];

export const initialConfig: ITaggerConfig = {
  blacklist: DEFAULT_BLACKLIST,
  showMales: false,
  mode: "auto",
  setTags: false,
  createTags: false,
  tagOperation: "merge",
  fingerprintQueue: {},
  excludedPerformerFields: DEFAULT_EXCLUDED_PERFORMER_FIELDS,
  excludedSceneFields: DEFAULT_EXCLUDED_SCENE_FIELDS,
  setOrganized: false,
};

export type ParseMode = "auto" | "filename" | "dir" | "path" | "metadata";
export interface ITaggerConfig {
  blacklist: string[];
  showMales: boolean;
  mode: ParseMode;
  setTags: boolean;
  createTags: boolean;
  tagOperation: string;
  selectedEndpoint?: string;
  fingerprintQueue: Record<string, string[]>;
  excludedPerformerFields?: string[];
  excludedSceneFields?: string[];
  setOrganized: boolean;
}

export const PERFORMER_FIELDS = [
  "name",
  "aliases",
  "image",
  "gender",
  "birthdate",
  "ethnicity",
  "country",
  "eye_color",
  "height",
  "measurements",
  "fake_tits",
  "career_length",
  "tattoos",
  "piercings",
];

export const SCENE_FIELDS = [
  "title",
  "url",
  "date",
  "description",
  "cover",
  "studio",
  "performers",
];
