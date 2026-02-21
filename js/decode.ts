export interface CaptionSpan {
  text: string;
  fgColor: string;
  bgColor: string;
  fgOpacity: number;
  bgOpacity: number;
  italic: boolean;
  underline: boolean;
  flash: boolean;
  penSize: number;
  fontTag: number;
  offset: number;
  edgeType: number;
  edgeColor: string;
}

export interface CaptionRow {
  row: number;
  spans: CaptionSpan[];
}

export interface CaptionRegion {
  id: number;
  justify: number;
  scrollDirection: number;
  printDirection: number;
  wordWrap: boolean;
  fillColor: string;
  fillOpacity: number;
  borderColor: string;
  borderType: number;
  anchorV: number;
  anchorH: number;
  anchorID: number;
  relativeToggle: boolean;
  priority: number;
  rows: CaptionRow[];
}

export interface CaptionData {
  channel: number;
  text?: string;
  regions: CaptionRegion[];
}

const MAGIC = 0xcc02;

function rgbToHex(r: number, g: number, b: number): string {
  return (
    r.toString(16).padStart(2, "0") +
    g.toString(16).padStart(2, "0") +
    b.toString(16).padStart(2, "0")
  );
}

export function parseCaptionData(data: Uint8Array): CaptionData | null {
  if (data.length < 2) return null;

  const magic = (data[0] << 8) | data[1];
  if (magic !== MAGIC) {
    return {
      channel: data[0],
      text: new TextDecoder().decode(data.slice(1)),
      regions: [],
    };
  }

  if (data.length < 5) return null;

  const version = data[2];
  const channel = data[3];
  const regionCount = data[4];
  let pos = 5;

  const regions: CaptionRegion[] = [];

  for (let i = 0; i < regionCount && pos < data.length; i++) {
    if (pos + 3 > data.length) break;

    const id = data[pos++];
    const flags = data[pos++];
    const fillFlags = data[pos++];

    const region: CaptionRegion = {
      id,
      justify: flags & 0x03,
      scrollDirection: (flags >> 2) & 0x03,
      printDirection: (flags >> 4) & 0x03,
      wordWrap: (flags & 0x40) !== 0,
      relativeToggle: (flags & 0x80) !== 0,
      fillOpacity: (fillFlags >> 6) & 0x03,
      borderType: (fillFlags >> 3) & 0x07,
      priority: fillFlags & 0x07,
      fillColor: "000000",
      borderColor: "000000",
      anchorV: 0,
      anchorH: 0,
      anchorID: 0,
      rows: [],
    };

    if (version >= 2) {
      if (pos + 9 > data.length) break;
      region.fillColor = rgbToHex(data[pos], data[pos + 1], data[pos + 2]);
      pos += 3;
      region.borderColor = rgbToHex(data[pos], data[pos + 1], data[pos + 2]);
      pos += 3;
      region.anchorV = data[pos++];
      region.anchorH = data[pos++];
      region.anchorID = data[pos++];
    }

    if (pos + 2 > data.length) break;
    const rowCount = (data[pos] << 8) | data[pos + 1];
    pos += 2;

    for (let r = 0; r < rowCount && pos < data.length; r++) {
      if (pos + 2 > data.length) break;
      const rowIdx = data[pos++];
      const spanCount = data[pos++];

      const row: CaptionRow = { row: rowIdx, spans: [] };

      for (let s = 0; s < spanCount && pos < data.length; s++) {
        if (pos + 2 > data.length) break;
        const textLen = (data[pos] << 8) | data[pos + 1];
        pos += 2;

        if (pos + textLen > data.length) break;
        const text = new TextDecoder().decode(data.slice(pos, pos + textLen));
        pos += textLen;

        if (pos + 9 > data.length) break;
        const fgColor = rgbToHex(data[pos], data[pos + 1], data[pos + 2]);
        pos += 3;
        const bgColor = rgbToHex(data[pos], data[pos + 1], data[pos + 2]);
        pos += 3;

        const attr0 = data[pos++];
        const attr1 = data[pos++];
        const attr2 = data[pos++];

        if (pos + 3 > data.length) break;
        const edgeColor = rgbToHex(data[pos], data[pos + 1], data[pos + 2]);
        pos += 3;

        const span: CaptionSpan = {
          text,
          fgColor,
          bgColor,
          fgOpacity: (attr0 >> 6) & 0x03,
          bgOpacity: (attr0 >> 4) & 0x03,
          italic: (attr0 & 0x08) !== 0,
          underline: (attr0 & 0x04) !== 0,
          flash: (attr0 & 0x02) !== 0,
          penSize: ((attr0 & 0x01) << 1) | ((attr1 >> 7) & 0x01),
          fontTag: (attr1 >> 4) & 0x07,
          offset: (attr1 >> 2) & 0x03,
          edgeType: (attr1 & 0x03) | (((attr2 >> 7) & 0x01) << 2),
          edgeColor,
        };

        row.spans.push(span);
      }

      region.rows.push(row);
    }

    regions.push(region);
  }

  return { channel, regions };
}
