import { SearchScene_searchScene_performers_performer_urls as URL } from 'src/definitions-box/SearchScene';

const CDN = 'https://cdn.stashdb.org';

export const sortImageURLs = (urls: URL[], orientation: 'portrait'|'landscape') => (
  urls.filter((u) => u.type === 'PHOTO').map((u:URL) => ({
      url: u.image_id ? `${CDN}/${u.image_id.slice(0, 2)}/${u.image_id.slice(2, 4)}/${u.image_id}` : u.url,
      width: u.width ?? 1,
      height: u.height ?? 1,
      aspect: orientation === 'portrait' ? (u.height ?? 1) / (u?.width ?? 1) > 1 : ((u.width ?? 1) / (u.height ?? 1)) > 1
  })).sort((a, b) => {
      if (a.aspect > b.aspect) return -1;
      if (a.aspect < b.aspect) return 1;
      if (orientation === 'portrait' && a.height > b.height) return -1;
      if (orientation === 'portrait' && a.height < b.height) return 1;
      if (orientation === 'landscape' && a.width > b.width) return -1;
      if (orientation === 'landscape' && a.width < b.width) return 1;
      return 0;
  })
)

export const getUrlByType = (
    urls:(URL|null)[],
    type:string,
    orientation?: 'portrait'|'landscape'
) => {
  if (urls.length > 0 && type === 'PHOTO' && orientation)
    return sortImageURLs(urls.filter(u => u !== null) as URL[], orientation)[0].url;
  return (urls && (urls.find((url) => url?.type === type) || {}).url) || '';
};