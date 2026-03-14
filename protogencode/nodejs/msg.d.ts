import * as $protobuf from "protobufjs";
import Long = require("long");
/** Properties of a MarketLevel. */
export interface IMarketLevel {

    /** MarketLevel price */
    price?: (google.protobuf.IInt64Value|null);

    /** MarketLevel qty */
    qty?: (google.protobuf.IUInt32Value|null);

    /** MarketLevel nord */
    nord?: (google.protobuf.IUInt32Value|null);

    /** MarketLevel num */
    num?: (google.protobuf.IUInt32Value|null);
}

/** Represents a MarketLevel. */
export class MarketLevel implements IMarketLevel {

    /**
     * Constructs a new MarketLevel.
     * @param [properties] Properties to set
     */
    constructor(properties?: IMarketLevel);

    /** MarketLevel price. */
    public price?: (google.protobuf.IInt64Value|null);

    /** MarketLevel qty. */
    public qty?: (google.protobuf.IUInt32Value|null);

    /** MarketLevel nord. */
    public nord?: (google.protobuf.IUInt32Value|null);

    /** MarketLevel num. */
    public num?: (google.protobuf.IUInt32Value|null);

    /**
     * Creates a new MarketLevel instance using the specified properties.
     * @param [properties] Properties to set
     * @returns MarketLevel instance
     */
    public static create(properties?: IMarketLevel): MarketLevel;

    /**
     * Encodes the specified MarketLevel message. Does not implicitly {@link MarketLevel.verify|verify} messages.
     * @param message MarketLevel message or plain object to encode
     * @param [writer] Writer to encode to
     * @returns Writer
     */
    public static encode(message: IMarketLevel, writer?: $protobuf.Writer): $protobuf.Writer;

    /**
     * Encodes the specified MarketLevel message, length delimited. Does not implicitly {@link MarketLevel.verify|verify} messages.
     * @param message MarketLevel message or plain object to encode
     * @param [writer] Writer to encode to
     * @returns Writer
     */
    public static encodeDelimited(message: IMarketLevel, writer?: $protobuf.Writer): $protobuf.Writer;

    /**
     * Decodes a MarketLevel message from the specified reader or buffer.
     * @param reader Reader or buffer to decode from
     * @param [length] Message length if known beforehand
     * @returns MarketLevel
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): MarketLevel;

    /**
     * Decodes a MarketLevel message from the specified reader or buffer, length delimited.
     * @param reader Reader or buffer to decode from
     * @returns MarketLevel
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): MarketLevel;

    /**
     * Verifies a MarketLevel message.
     * @param message Plain object to verify
     * @returns `null` if valid, otherwise the reason why it is not
     */
    public static verify(message: { [k: string]: any }): (string|null);

    /**
     * Creates a MarketLevel message from a plain object. Also converts values to their respective internal types.
     * @param object Plain object
     * @returns MarketLevel
     */
    public static fromObject(object: { [k: string]: any }): MarketLevel;

    /**
     * Creates a plain object from a MarketLevel message. Also converts values to other types if specified.
     * @param message MarketLevel
     * @param [options] Conversion options
     * @returns Plain object
     */
    public static toObject(message: MarketLevel, options?: $protobuf.IConversionOptions): { [k: string]: any };

    /**
     * Converts this MarketLevel to JSON.
     * @returns JSON object
     */
    public toJSON(): { [k: string]: any };

    /**
     * Gets the default type url for MarketLevel
     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
     * @returns The default type url
     */
    public static getTypeUrl(typeUrlPrefix?: string): string;
}

/** Properties of a Depth. */
export interface IDepth {

    /** Depth tbq */
    tbq?: (google.protobuf.IUInt64Value|null);

    /** Depth tsq */
    tsq?: (google.protobuf.IUInt64Value|null);

    /** Depth asks */
    asks?: (IMarketLevel[]|null);

    /** Depth bids */
    bids?: (IMarketLevel[]|null);
}

/** Represents a Depth. */
export class Depth implements IDepth {

    /**
     * Constructs a new Depth.
     * @param [properties] Properties to set
     */
    constructor(properties?: IDepth);

    /** Depth tbq. */
    public tbq?: (google.protobuf.IUInt64Value|null);

    /** Depth tsq. */
    public tsq?: (google.protobuf.IUInt64Value|null);

    /** Depth asks. */
    public asks: IMarketLevel[];

    /** Depth bids. */
    public bids: IMarketLevel[];

    /**
     * Creates a new Depth instance using the specified properties.
     * @param [properties] Properties to set
     * @returns Depth instance
     */
    public static create(properties?: IDepth): Depth;

    /**
     * Encodes the specified Depth message. Does not implicitly {@link Depth.verify|verify} messages.
     * @param message Depth message or plain object to encode
     * @param [writer] Writer to encode to
     * @returns Writer
     */
    public static encode(message: IDepth, writer?: $protobuf.Writer): $protobuf.Writer;

    /**
     * Encodes the specified Depth message, length delimited. Does not implicitly {@link Depth.verify|verify} messages.
     * @param message Depth message or plain object to encode
     * @param [writer] Writer to encode to
     * @returns Writer
     */
    public static encodeDelimited(message: IDepth, writer?: $protobuf.Writer): $protobuf.Writer;

    /**
     * Decodes a Depth message from the specified reader or buffer.
     * @param reader Reader or buffer to decode from
     * @param [length] Message length if known beforehand
     * @returns Depth
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): Depth;

    /**
     * Decodes a Depth message from the specified reader or buffer, length delimited.
     * @param reader Reader or buffer to decode from
     * @returns Depth
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): Depth;

    /**
     * Verifies a Depth message.
     * @param message Plain object to verify
     * @returns `null` if valid, otherwise the reason why it is not
     */
    public static verify(message: { [k: string]: any }): (string|null);

    /**
     * Creates a Depth message from a plain object. Also converts values to their respective internal types.
     * @param object Plain object
     * @returns Depth
     */
    public static fromObject(object: { [k: string]: any }): Depth;

    /**
     * Creates a plain object from a Depth message. Also converts values to other types if specified.
     * @param message Depth
     * @param [options] Conversion options
     * @returns Plain object
     */
    public static toObject(message: Depth, options?: $protobuf.IConversionOptions): { [k: string]: any };

    /**
     * Converts this Depth to JSON.
     * @returns JSON object
     */
    public toJSON(): { [k: string]: any };

    /**
     * Gets the default type url for Depth
     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
     * @returns The default type url
     */
    public static getTypeUrl(typeUrlPrefix?: string): string;
}

/** Properties of a Quote. */
export interface IQuote {

    /** Quote ltp */
    ltp?: (google.protobuf.IInt64Value|null);

    /** Quote ltt */
    ltt?: (google.protobuf.IUInt32Value|null);

    /** Quote ltq */
    ltq?: (google.protobuf.IUInt32Value|null);

    /** Quote vtt */
    vtt?: (google.protobuf.IUInt64Value|null);

    /** Quote vttDiff */
    vttDiff?: (google.protobuf.IUInt64Value|null);

    /** Quote oi */
    oi?: (google.protobuf.IUInt64Value|null);

    /** Quote ltpc */
    ltpc?: (google.protobuf.IInt64Value|null);
}

/** Represents a Quote. */
export class Quote implements IQuote {

    /**
     * Constructs a new Quote.
     * @param [properties] Properties to set
     */
    constructor(properties?: IQuote);

    /** Quote ltp. */
    public ltp?: (google.protobuf.IInt64Value|null);

    /** Quote ltt. */
    public ltt?: (google.protobuf.IUInt32Value|null);

    /** Quote ltq. */
    public ltq?: (google.protobuf.IUInt32Value|null);

    /** Quote vtt. */
    public vtt?: (google.protobuf.IUInt64Value|null);

    /** Quote vttDiff. */
    public vttDiff?: (google.protobuf.IUInt64Value|null);

    /** Quote oi. */
    public oi?: (google.protobuf.IUInt64Value|null);

    /** Quote ltpc. */
    public ltpc?: (google.protobuf.IInt64Value|null);

    /**
     * Creates a new Quote instance using the specified properties.
     * @param [properties] Properties to set
     * @returns Quote instance
     */
    public static create(properties?: IQuote): Quote;

    /**
     * Encodes the specified Quote message. Does not implicitly {@link Quote.verify|verify} messages.
     * @param message Quote message or plain object to encode
     * @param [writer] Writer to encode to
     * @returns Writer
     */
    public static encode(message: IQuote, writer?: $protobuf.Writer): $protobuf.Writer;

    /**
     * Encodes the specified Quote message, length delimited. Does not implicitly {@link Quote.verify|verify} messages.
     * @param message Quote message or plain object to encode
     * @param [writer] Writer to encode to
     * @returns Writer
     */
    public static encodeDelimited(message: IQuote, writer?: $protobuf.Writer): $protobuf.Writer;

    /**
     * Decodes a Quote message from the specified reader or buffer.
     * @param reader Reader or buffer to decode from
     * @param [length] Message length if known beforehand
     * @returns Quote
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): Quote;

    /**
     * Decodes a Quote message from the specified reader or buffer, length delimited.
     * @param reader Reader or buffer to decode from
     * @returns Quote
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): Quote;

    /**
     * Verifies a Quote message.
     * @param message Plain object to verify
     * @returns `null` if valid, otherwise the reason why it is not
     */
    public static verify(message: { [k: string]: any }): (string|null);

    /**
     * Creates a Quote message from a plain object. Also converts values to their respective internal types.
     * @param object Plain object
     * @returns Quote
     */
    public static fromObject(object: { [k: string]: any }): Quote;

    /**
     * Creates a plain object from a Quote message. Also converts values to other types if specified.
     * @param message Quote
     * @param [options] Conversion options
     * @returns Plain object
     */
    public static toObject(message: Quote, options?: $protobuf.IConversionOptions): { [k: string]: any };

    /**
     * Converts this Quote to JSON.
     * @returns JSON object
     */
    public toJSON(): { [k: string]: any };

    /**
     * Gets the default type url for Quote
     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
     * @returns The default type url
     */
    public static getTypeUrl(typeUrlPrefix?: string): string;
}

/** Properties of an ExtendedQuote. */
export interface IExtendedQuote {

    /** ExtendedQuote atp */
    atp?: (google.protobuf.IInt64Value|null);

    /** ExtendedQuote cp */
    cp?: (google.protobuf.IInt64Value|null);

    /** ExtendedQuote lc */
    lc?: (google.protobuf.IUInt32Value|null);

    /** ExtendedQuote uc */
    uc?: (google.protobuf.IUInt32Value|null);

    /** ExtendedQuote yh */
    yh?: (google.protobuf.IInt64Value|null);

    /** ExtendedQuote yl */
    yl?: (google.protobuf.IInt64Value|null);

    /** ExtendedQuote poi */
    poi?: (google.protobuf.IUInt64Value|null);

    /** ExtendedQuote oich */
    oich?: (google.protobuf.IInt64Value|null);

    /** ExtendedQuote pc */
    pc?: (google.protobuf.IUInt32Value|null);
}

/** Represents an ExtendedQuote. */
export class ExtendedQuote implements IExtendedQuote {

    /**
     * Constructs a new ExtendedQuote.
     * @param [properties] Properties to set
     */
    constructor(properties?: IExtendedQuote);

    /** ExtendedQuote atp. */
    public atp?: (google.protobuf.IInt64Value|null);

    /** ExtendedQuote cp. */
    public cp?: (google.protobuf.IInt64Value|null);

    /** ExtendedQuote lc. */
    public lc?: (google.protobuf.IUInt32Value|null);

    /** ExtendedQuote uc. */
    public uc?: (google.protobuf.IUInt32Value|null);

    /** ExtendedQuote yh. */
    public yh?: (google.protobuf.IInt64Value|null);

    /** ExtendedQuote yl. */
    public yl?: (google.protobuf.IInt64Value|null);

    /** ExtendedQuote poi. */
    public poi?: (google.protobuf.IUInt64Value|null);

    /** ExtendedQuote oich. */
    public oich?: (google.protobuf.IInt64Value|null);

    /** ExtendedQuote pc. */
    public pc?: (google.protobuf.IUInt32Value|null);

    /**
     * Creates a new ExtendedQuote instance using the specified properties.
     * @param [properties] Properties to set
     * @returns ExtendedQuote instance
     */
    public static create(properties?: IExtendedQuote): ExtendedQuote;

    /**
     * Encodes the specified ExtendedQuote message. Does not implicitly {@link ExtendedQuote.verify|verify} messages.
     * @param message ExtendedQuote message or plain object to encode
     * @param [writer] Writer to encode to
     * @returns Writer
     */
    public static encode(message: IExtendedQuote, writer?: $protobuf.Writer): $protobuf.Writer;

    /**
     * Encodes the specified ExtendedQuote message, length delimited. Does not implicitly {@link ExtendedQuote.verify|verify} messages.
     * @param message ExtendedQuote message or plain object to encode
     * @param [writer] Writer to encode to
     * @returns Writer
     */
    public static encodeDelimited(message: IExtendedQuote, writer?: $protobuf.Writer): $protobuf.Writer;

    /**
     * Decodes an ExtendedQuote message from the specified reader or buffer.
     * @param reader Reader or buffer to decode from
     * @param [length] Message length if known beforehand
     * @returns ExtendedQuote
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ExtendedQuote;

    /**
     * Decodes an ExtendedQuote message from the specified reader or buffer, length delimited.
     * @param reader Reader or buffer to decode from
     * @returns ExtendedQuote
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ExtendedQuote;

    /**
     * Verifies an ExtendedQuote message.
     * @param message Plain object to verify
     * @returns `null` if valid, otherwise the reason why it is not
     */
    public static verify(message: { [k: string]: any }): (string|null);

    /**
     * Creates an ExtendedQuote message from a plain object. Also converts values to their respective internal types.
     * @param object Plain object
     * @returns ExtendedQuote
     */
    public static fromObject(object: { [k: string]: any }): ExtendedQuote;

    /**
     * Creates a plain object from an ExtendedQuote message. Also converts values to other types if specified.
     * @param message ExtendedQuote
     * @param [options] Conversion options
     * @returns Plain object
     */
    public static toObject(message: ExtendedQuote, options?: $protobuf.IConversionOptions): { [k: string]: any };

    /**
     * Converts this ExtendedQuote to JSON.
     * @returns JSON object
     */
    public toJSON(): { [k: string]: any };

    /**
     * Gets the default type url for ExtendedQuote
     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
     * @returns The default type url
     */
    public static getTypeUrl(typeUrlPrefix?: string): string;
}

/** Properties of a DailyQuote. */
export interface IDailyQuote {

    /** DailyQuote do */
    "do"?: (google.protobuf.IInt64Value|null);

    /** DailyQuote dh */
    dh?: (google.protobuf.IInt64Value|null);

    /** DailyQuote dl */
    dl?: (google.protobuf.IInt64Value|null);

    /** DailyQuote dc */
    dc?: (google.protobuf.IInt64Value|null);

    /** DailyQuote dhoi */
    dhoi?: (google.protobuf.IUInt64Value|null);

    /** DailyQuote dloi */
    dloi?: (google.protobuf.IUInt64Value|null);
}

/** Represents a DailyQuote. */
export class DailyQuote implements IDailyQuote {

    /**
     * Constructs a new DailyQuote.
     * @param [properties] Properties to set
     */
    constructor(properties?: IDailyQuote);

    /** DailyQuote do. */
    public do?: (google.protobuf.IInt64Value|null);

    /** DailyQuote dh. */
    public dh?: (google.protobuf.IInt64Value|null);

    /** DailyQuote dl. */
    public dl?: (google.protobuf.IInt64Value|null);

    /** DailyQuote dc. */
    public dc?: (google.protobuf.IInt64Value|null);

    /** DailyQuote dhoi. */
    public dhoi?: (google.protobuf.IUInt64Value|null);

    /** DailyQuote dloi. */
    public dloi?: (google.protobuf.IUInt64Value|null);

    /**
     * Creates a new DailyQuote instance using the specified properties.
     * @param [properties] Properties to set
     * @returns DailyQuote instance
     */
    public static create(properties?: IDailyQuote): DailyQuote;

    /**
     * Encodes the specified DailyQuote message. Does not implicitly {@link DailyQuote.verify|verify} messages.
     * @param message DailyQuote message or plain object to encode
     * @param [writer] Writer to encode to
     * @returns Writer
     */
    public static encode(message: IDailyQuote, writer?: $protobuf.Writer): $protobuf.Writer;

    /**
     * Encodes the specified DailyQuote message, length delimited. Does not implicitly {@link DailyQuote.verify|verify} messages.
     * @param message DailyQuote message or plain object to encode
     * @param [writer] Writer to encode to
     * @returns Writer
     */
    public static encodeDelimited(message: IDailyQuote, writer?: $protobuf.Writer): $protobuf.Writer;

    /**
     * Decodes a DailyQuote message from the specified reader or buffer.
     * @param reader Reader or buffer to decode from
     * @param [length] Message length if known beforehand
     * @returns DailyQuote
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): DailyQuote;

    /**
     * Decodes a DailyQuote message from the specified reader or buffer, length delimited.
     * @param reader Reader or buffer to decode from
     * @returns DailyQuote
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): DailyQuote;

    /**
     * Verifies a DailyQuote message.
     * @param message Plain object to verify
     * @returns `null` if valid, otherwise the reason why it is not
     */
    public static verify(message: { [k: string]: any }): (string|null);

    /**
     * Creates a DailyQuote message from a plain object. Also converts values to their respective internal types.
     * @param object Plain object
     * @returns DailyQuote
     */
    public static fromObject(object: { [k: string]: any }): DailyQuote;

    /**
     * Creates a plain object from a DailyQuote message. Also converts values to other types if specified.
     * @param message DailyQuote
     * @param [options] Conversion options
     * @returns Plain object
     */
    public static toObject(message: DailyQuote, options?: $protobuf.IConversionOptions): { [k: string]: any };

    /**
     * Converts this DailyQuote to JSON.
     * @returns JSON object
     */
    public toJSON(): { [k: string]: any };

    /**
     * Gets the default type url for DailyQuote
     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
     * @returns The default type url
     */
    public static getTypeUrl(typeUrlPrefix?: string): string;
}

/** Properties of a OHLCV. */
export interface IOHLCV {

    /** OHLCV open */
    open?: (google.protobuf.IInt64Value|null);

    /** OHLCV high */
    high?: (google.protobuf.IInt64Value|null);

    /** OHLCV low */
    low?: (google.protobuf.IInt64Value|null);

    /** OHLCV close */
    close?: (google.protobuf.IInt64Value|null);

    /** OHLCV volume */
    volume?: (google.protobuf.IUInt32Value|null);

    /** OHLCV epoch */
    epoch?: (google.protobuf.IUInt32Value|null);
}

/** Represents a OHLCV. */
export class OHLCV implements IOHLCV {

    /**
     * Constructs a new OHLCV.
     * @param [properties] Properties to set
     */
    constructor(properties?: IOHLCV);

    /** OHLCV open. */
    public open?: (google.protobuf.IInt64Value|null);

    /** OHLCV high. */
    public high?: (google.protobuf.IInt64Value|null);

    /** OHLCV low. */
    public low?: (google.protobuf.IInt64Value|null);

    /** OHLCV close. */
    public close?: (google.protobuf.IInt64Value|null);

    /** OHLCV volume. */
    public volume?: (google.protobuf.IUInt32Value|null);

    /** OHLCV epoch. */
    public epoch?: (google.protobuf.IUInt32Value|null);

    /**
     * Creates a new OHLCV instance using the specified properties.
     * @param [properties] Properties to set
     * @returns OHLCV instance
     */
    public static create(properties?: IOHLCV): OHLCV;

    /**
     * Encodes the specified OHLCV message. Does not implicitly {@link OHLCV.verify|verify} messages.
     * @param message OHLCV message or plain object to encode
     * @param [writer] Writer to encode to
     * @returns Writer
     */
    public static encode(message: IOHLCV, writer?: $protobuf.Writer): $protobuf.Writer;

    /**
     * Encodes the specified OHLCV message, length delimited. Does not implicitly {@link OHLCV.verify|verify} messages.
     * @param message OHLCV message or plain object to encode
     * @param [writer] Writer to encode to
     * @returns Writer
     */
    public static encodeDelimited(message: IOHLCV, writer?: $protobuf.Writer): $protobuf.Writer;

    /**
     * Decodes a OHLCV message from the specified reader or buffer.
     * @param reader Reader or buffer to decode from
     * @param [length] Message length if known beforehand
     * @returns OHLCV
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): OHLCV;

    /**
     * Decodes a OHLCV message from the specified reader or buffer, length delimited.
     * @param reader Reader or buffer to decode from
     * @returns OHLCV
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): OHLCV;

    /**
     * Verifies a OHLCV message.
     * @param message Plain object to verify
     * @returns `null` if valid, otherwise the reason why it is not
     */
    public static verify(message: { [k: string]: any }): (string|null);

    /**
     * Creates a OHLCV message from a plain object. Also converts values to their respective internal types.
     * @param object Plain object
     * @returns OHLCV
     */
    public static fromObject(object: { [k: string]: any }): OHLCV;

    /**
     * Creates a plain object from a OHLCV message. Also converts values to other types if specified.
     * @param message OHLCV
     * @param [options] Conversion options
     * @returns Plain object
     */
    public static toObject(message: OHLCV, options?: $protobuf.IConversionOptions): { [k: string]: any };

    /**
     * Converts this OHLCV to JSON.
     * @returns JSON object
     */
    public toJSON(): { [k: string]: any };

    /**
     * Gets the default type url for OHLCV
     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
     * @returns The default type url
     */
    public static getTypeUrl(typeUrlPrefix?: string): string;
}

/** Properties of a SymDetail. */
export interface ISymDetail {

    /** SymDetail ticksize */
    ticksize?: (string|null);
}

/** Represents a SymDetail. */
export class SymDetail implements ISymDetail {

    /**
     * Constructs a new SymDetail.
     * @param [properties] Properties to set
     */
    constructor(properties?: ISymDetail);

    /** SymDetail ticksize. */
    public ticksize: string;

    /**
     * Creates a new SymDetail instance using the specified properties.
     * @param [properties] Properties to set
     * @returns SymDetail instance
     */
    public static create(properties?: ISymDetail): SymDetail;

    /**
     * Encodes the specified SymDetail message. Does not implicitly {@link SymDetail.verify|verify} messages.
     * @param message SymDetail message or plain object to encode
     * @param [writer] Writer to encode to
     * @returns Writer
     */
    public static encode(message: ISymDetail, writer?: $protobuf.Writer): $protobuf.Writer;

    /**
     * Encodes the specified SymDetail message, length delimited. Does not implicitly {@link SymDetail.verify|verify} messages.
     * @param message SymDetail message or plain object to encode
     * @param [writer] Writer to encode to
     * @returns Writer
     */
    public static encodeDelimited(message: ISymDetail, writer?: $protobuf.Writer): $protobuf.Writer;

    /**
     * Decodes a SymDetail message from the specified reader or buffer.
     * @param reader Reader or buffer to decode from
     * @param [length] Message length if known beforehand
     * @returns SymDetail
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): SymDetail;

    /**
     * Decodes a SymDetail message from the specified reader or buffer, length delimited.
     * @param reader Reader or buffer to decode from
     * @returns SymDetail
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): SymDetail;

    /**
     * Verifies a SymDetail message.
     * @param message Plain object to verify
     * @returns `null` if valid, otherwise the reason why it is not
     */
    public static verify(message: { [k: string]: any }): (string|null);

    /**
     * Creates a SymDetail message from a plain object. Also converts values to their respective internal types.
     * @param object Plain object
     * @returns SymDetail
     */
    public static fromObject(object: { [k: string]: any }): SymDetail;

    /**
     * Creates a plain object from a SymDetail message. Also converts values to other types if specified.
     * @param message SymDetail
     * @param [options] Conversion options
     * @returns Plain object
     */
    public static toObject(message: SymDetail, options?: $protobuf.IConversionOptions): { [k: string]: any };

    /**
     * Converts this SymDetail to JSON.
     * @returns JSON object
     */
    public toJSON(): { [k: string]: any };

    /**
     * Gets the default type url for SymDetail
     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
     * @returns The default type url
     */
    public static getTypeUrl(typeUrlPrefix?: string): string;
}

/** MessageType enum. */
export enum MessageType {
    ping = 0,
    quote = 1,
    extended_quote = 2,
    daily_quote = 3,
    market_level = 4,
    ohlcv = 5,
    depth = 6,
    all = 7,
    response = 8
}

/** Represents a MarketFeed. */
export class MarketFeed implements IMarketFeed {

    /**
     * Constructs a new MarketFeed.
     * @param [properties] Properties to set
     */
    constructor(properties?: IMarketFeed);

    /** MarketFeed quote. */
    public quote?: (IQuote|null);

    /** MarketFeed eq. */
    public eq?: (IExtendedQuote|null);

    /** MarketFeed dq. */
    public dq?: (IDailyQuote|null);

    /** MarketFeed ohlcv. */
    public ohlcv?: (IOHLCV|null);

    /** MarketFeed depth. */
    public depth?: (IDepth|null);

    /** MarketFeed feedTime. */
    public feedTime?: (google.protobuf.IUInt64Value|null);

    /** MarketFeed sendTime. */
    public sendTime?: (google.protobuf.IUInt64Value|null);

    /** MarketFeed token. */
    public token: string;

    /** MarketFeed sequenceNo. */
    public sequenceNo: (number|Long);

    /** MarketFeed snapshot. */
    public snapshot: boolean;

    /** MarketFeed ticker. */
    public ticker: string;

    /** MarketFeed symdetail. */
    public symdetail?: (ISymDetail|null);

    /**
     * Creates a new MarketFeed instance using the specified properties.
     * @param [properties] Properties to set
     * @returns MarketFeed instance
     */
    public static create(properties?: IMarketFeed): MarketFeed;

    /**
     * Encodes the specified MarketFeed message. Does not implicitly {@link MarketFeed.verify|verify} messages.
     * @param message MarketFeed message or plain object to encode
     * @param [writer] Writer to encode to
     * @returns Writer
     */
    public static encode(message: IMarketFeed, writer?: $protobuf.Writer): $protobuf.Writer;

    /**
     * Encodes the specified MarketFeed message, length delimited. Does not implicitly {@link MarketFeed.verify|verify} messages.
     * @param message MarketFeed message or plain object to encode
     * @param [writer] Writer to encode to
     * @returns Writer
     */
    public static encodeDelimited(message: IMarketFeed, writer?: $protobuf.Writer): $protobuf.Writer;

    /**
     * Decodes a MarketFeed message from the specified reader or buffer.
     * @param reader Reader or buffer to decode from
     * @param [length] Message length if known beforehand
     * @returns MarketFeed
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): MarketFeed;

    /**
     * Decodes a MarketFeed message from the specified reader or buffer, length delimited.
     * @param reader Reader or buffer to decode from
     * @returns MarketFeed
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): MarketFeed;

    /**
     * Verifies a MarketFeed message.
     * @param message Plain object to verify
     * @returns `null` if valid, otherwise the reason why it is not
     */
    public static verify(message: { [k: string]: any }): (string|null);

    /**
     * Creates a MarketFeed message from a plain object. Also converts values to their respective internal types.
     * @param object Plain object
     * @returns MarketFeed
     */
    public static fromObject(object: { [k: string]: any }): MarketFeed;

    /**
     * Creates a plain object from a MarketFeed message. Also converts values to other types if specified.
     * @param message MarketFeed
     * @param [options] Conversion options
     * @returns Plain object
     */
    public static toObject(message: MarketFeed, options?: $protobuf.IConversionOptions): { [k: string]: any };

    /**
     * Converts this MarketFeed to JSON.
     * @returns JSON object
     */
    public toJSON(): { [k: string]: any };

    /**
     * Gets the default type url for MarketFeed
     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
     * @returns The default type url
     */
    public static getTypeUrl(typeUrlPrefix?: string): string;
}

/** Represents a SocketMessage. */
export class SocketMessage implements ISocketMessage {

    /**
     * Constructs a new SocketMessage.
     * @param [properties] Properties to set
     */
    constructor(properties?: ISocketMessage);

    /** SocketMessage type. */
    public type: MessageType;

    /** SocketMessage feeds. */
    public feeds: { [k: string]: IMarketFeed };

    /** SocketMessage snapshot. */
    public snapshot: boolean;

    /** SocketMessage msg. */
    public msg: string;

    /** SocketMessage error. */
    public error: boolean;

    /**
     * Creates a new SocketMessage instance using the specified properties.
     * @param [properties] Properties to set
     * @returns SocketMessage instance
     */
    public static create(properties?: ISocketMessage): SocketMessage;

    /**
     * Encodes the specified SocketMessage message. Does not implicitly {@link SocketMessage.verify|verify} messages.
     * @param message SocketMessage message or plain object to encode
     * @param [writer] Writer to encode to
     * @returns Writer
     */
    public static encode(message: ISocketMessage, writer?: $protobuf.Writer): $protobuf.Writer;

    /**
     * Encodes the specified SocketMessage message, length delimited. Does not implicitly {@link SocketMessage.verify|verify} messages.
     * @param message SocketMessage message or plain object to encode
     * @param [writer] Writer to encode to
     * @returns Writer
     */
    public static encodeDelimited(message: ISocketMessage, writer?: $protobuf.Writer): $protobuf.Writer;

    /**
     * Decodes a SocketMessage message from the specified reader or buffer.
     * @param reader Reader or buffer to decode from
     * @param [length] Message length if known beforehand
     * @returns SocketMessage
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): SocketMessage;

    /**
     * Decodes a SocketMessage message from the specified reader or buffer, length delimited.
     * @param reader Reader or buffer to decode from
     * @returns SocketMessage
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): SocketMessage;

    /**
     * Verifies a SocketMessage message.
     * @param message Plain object to verify
     * @returns `null` if valid, otherwise the reason why it is not
     */
    public static verify(message: { [k: string]: any }): (string|null);

    /**
     * Creates a SocketMessage message from a plain object. Also converts values to their respective internal types.
     * @param object Plain object
     * @returns SocketMessage
     */
    public static fromObject(object: { [k: string]: any }): SocketMessage;

    /**
     * Creates a plain object from a SocketMessage message. Also converts values to other types if specified.
     * @param message SocketMessage
     * @param [options] Conversion options
     * @returns Plain object
     */
    public static toObject(message: SocketMessage, options?: $protobuf.IConversionOptions): { [k: string]: any };

    /**
     * Converts this SocketMessage to JSON.
     * @returns JSON object
     */
    public toJSON(): { [k: string]: any };

    /**
     * Gets the default type url for SocketMessage
     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
     * @returns The default type url
     */
    public static getTypeUrl(typeUrlPrefix?: string): string;
}

/** Namespace google. */
export namespace google {

    /** Namespace protobuf. */
    namespace protobuf {

        /** Properties of a DoubleValue. */
        interface IDoubleValue {

            /** DoubleValue value */
            value?: (number|null);
        }

        /** Represents a DoubleValue. */
        class DoubleValue implements IDoubleValue {

            /**
             * Constructs a new DoubleValue.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IDoubleValue);

            /** DoubleValue value. */
            public value: number;

            /**
             * Creates a new DoubleValue instance using the specified properties.
             * @param [properties] Properties to set
             * @returns DoubleValue instance
             */
            public static create(properties?: google.protobuf.IDoubleValue): google.protobuf.DoubleValue;

            /**
             * Encodes the specified DoubleValue message. Does not implicitly {@link google.protobuf.DoubleValue.verify|verify} messages.
             * @param message DoubleValue message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IDoubleValue, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified DoubleValue message, length delimited. Does not implicitly {@link google.protobuf.DoubleValue.verify|verify} messages.
             * @param message DoubleValue message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IDoubleValue, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a DoubleValue message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns DoubleValue
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.DoubleValue;

            /**
             * Decodes a DoubleValue message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns DoubleValue
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.DoubleValue;

            /**
             * Verifies a DoubleValue message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a DoubleValue message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns DoubleValue
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.DoubleValue;

            /**
             * Creates a plain object from a DoubleValue message. Also converts values to other types if specified.
             * @param message DoubleValue
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.DoubleValue, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this DoubleValue to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for DoubleValue
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        /** Properties of a FloatValue. */
        interface IFloatValue {

            /** FloatValue value */
            value?: (number|null);
        }

        /** Represents a FloatValue. */
        class FloatValue implements IFloatValue {

            /**
             * Constructs a new FloatValue.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IFloatValue);

            /** FloatValue value. */
            public value: number;

            /**
             * Creates a new FloatValue instance using the specified properties.
             * @param [properties] Properties to set
             * @returns FloatValue instance
             */
            public static create(properties?: google.protobuf.IFloatValue): google.protobuf.FloatValue;

            /**
             * Encodes the specified FloatValue message. Does not implicitly {@link google.protobuf.FloatValue.verify|verify} messages.
             * @param message FloatValue message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IFloatValue, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified FloatValue message, length delimited. Does not implicitly {@link google.protobuf.FloatValue.verify|verify} messages.
             * @param message FloatValue message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IFloatValue, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a FloatValue message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns FloatValue
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.FloatValue;

            /**
             * Decodes a FloatValue message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns FloatValue
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.FloatValue;

            /**
             * Verifies a FloatValue message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a FloatValue message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns FloatValue
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.FloatValue;

            /**
             * Creates a plain object from a FloatValue message. Also converts values to other types if specified.
             * @param message FloatValue
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.FloatValue, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this FloatValue to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for FloatValue
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        /** Properties of an Int64Value. */
        interface IInt64Value {

            /** Int64Value value */
            value?: (number|Long|null);
        }

        /** Represents an Int64Value. */
        class Int64Value implements IInt64Value {

            /**
             * Constructs a new Int64Value.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IInt64Value);

            /** Int64Value value. */
            public value: (number|Long);

            /**
             * Creates a new Int64Value instance using the specified properties.
             * @param [properties] Properties to set
             * @returns Int64Value instance
             */
            public static create(properties?: google.protobuf.IInt64Value): google.protobuf.Int64Value;

            /**
             * Encodes the specified Int64Value message. Does not implicitly {@link google.protobuf.Int64Value.verify|verify} messages.
             * @param message Int64Value message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IInt64Value, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified Int64Value message, length delimited. Does not implicitly {@link google.protobuf.Int64Value.verify|verify} messages.
             * @param message Int64Value message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IInt64Value, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes an Int64Value message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns Int64Value
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.Int64Value;

            /**
             * Decodes an Int64Value message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns Int64Value
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.Int64Value;

            /**
             * Verifies an Int64Value message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates an Int64Value message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns Int64Value
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.Int64Value;

            /**
             * Creates a plain object from an Int64Value message. Also converts values to other types if specified.
             * @param message Int64Value
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.Int64Value, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this Int64Value to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for Int64Value
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        /** Properties of a UInt64Value. */
        interface IUInt64Value {

            /** UInt64Value value */
            value?: (number|Long|null);
        }

        /** Represents a UInt64Value. */
        class UInt64Value implements IUInt64Value {

            /**
             * Constructs a new UInt64Value.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IUInt64Value);

            /** UInt64Value value. */
            public value: (number|Long);

            /**
             * Creates a new UInt64Value instance using the specified properties.
             * @param [properties] Properties to set
             * @returns UInt64Value instance
             */
            public static create(properties?: google.protobuf.IUInt64Value): google.protobuf.UInt64Value;

            /**
             * Encodes the specified UInt64Value message. Does not implicitly {@link google.protobuf.UInt64Value.verify|verify} messages.
             * @param message UInt64Value message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IUInt64Value, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified UInt64Value message, length delimited. Does not implicitly {@link google.protobuf.UInt64Value.verify|verify} messages.
             * @param message UInt64Value message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IUInt64Value, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a UInt64Value message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns UInt64Value
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.UInt64Value;

            /**
             * Decodes a UInt64Value message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns UInt64Value
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.UInt64Value;

            /**
             * Verifies a UInt64Value message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a UInt64Value message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns UInt64Value
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.UInt64Value;

            /**
             * Creates a plain object from a UInt64Value message. Also converts values to other types if specified.
             * @param message UInt64Value
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.UInt64Value, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this UInt64Value to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for UInt64Value
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        /** Properties of an Int32Value. */
        interface IInt32Value {

            /** Int32Value value */
            value?: (number|null);
        }

        /** Represents an Int32Value. */
        class Int32Value implements IInt32Value {

            /**
             * Constructs a new Int32Value.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IInt32Value);

            /** Int32Value value. */
            public value: number;

            /**
             * Creates a new Int32Value instance using the specified properties.
             * @param [properties] Properties to set
             * @returns Int32Value instance
             */
            public static create(properties?: google.protobuf.IInt32Value): google.protobuf.Int32Value;

            /**
             * Encodes the specified Int32Value message. Does not implicitly {@link google.protobuf.Int32Value.verify|verify} messages.
             * @param message Int32Value message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IInt32Value, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified Int32Value message, length delimited. Does not implicitly {@link google.protobuf.Int32Value.verify|verify} messages.
             * @param message Int32Value message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IInt32Value, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes an Int32Value message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns Int32Value
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.Int32Value;

            /**
             * Decodes an Int32Value message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns Int32Value
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.Int32Value;

            /**
             * Verifies an Int32Value message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates an Int32Value message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns Int32Value
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.Int32Value;

            /**
             * Creates a plain object from an Int32Value message. Also converts values to other types if specified.
             * @param message Int32Value
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.Int32Value, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this Int32Value to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for Int32Value
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        /** Properties of a UInt32Value. */
        interface IUInt32Value {

            /** UInt32Value value */
            value?: (number|null);
        }

        /** Represents a UInt32Value. */
        class UInt32Value implements IUInt32Value {

            /**
             * Constructs a new UInt32Value.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IUInt32Value);

            /** UInt32Value value. */
            public value: number;

            /**
             * Creates a new UInt32Value instance using the specified properties.
             * @param [properties] Properties to set
             * @returns UInt32Value instance
             */
            public static create(properties?: google.protobuf.IUInt32Value): google.protobuf.UInt32Value;

            /**
             * Encodes the specified UInt32Value message. Does not implicitly {@link google.protobuf.UInt32Value.verify|verify} messages.
             * @param message UInt32Value message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IUInt32Value, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified UInt32Value message, length delimited. Does not implicitly {@link google.protobuf.UInt32Value.verify|verify} messages.
             * @param message UInt32Value message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IUInt32Value, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a UInt32Value message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns UInt32Value
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.UInt32Value;

            /**
             * Decodes a UInt32Value message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns UInt32Value
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.UInt32Value;

            /**
             * Verifies a UInt32Value message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a UInt32Value message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns UInt32Value
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.UInt32Value;

            /**
             * Creates a plain object from a UInt32Value message. Also converts values to other types if specified.
             * @param message UInt32Value
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.UInt32Value, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this UInt32Value to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for UInt32Value
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        /** Properties of a BoolValue. */
        interface IBoolValue {

            /** BoolValue value */
            value?: (boolean|null);
        }

        /** Represents a BoolValue. */
        class BoolValue implements IBoolValue {

            /**
             * Constructs a new BoolValue.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IBoolValue);

            /** BoolValue value. */
            public value: boolean;

            /**
             * Creates a new BoolValue instance using the specified properties.
             * @param [properties] Properties to set
             * @returns BoolValue instance
             */
            public static create(properties?: google.protobuf.IBoolValue): google.protobuf.BoolValue;

            /**
             * Encodes the specified BoolValue message. Does not implicitly {@link google.protobuf.BoolValue.verify|verify} messages.
             * @param message BoolValue message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IBoolValue, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified BoolValue message, length delimited. Does not implicitly {@link google.protobuf.BoolValue.verify|verify} messages.
             * @param message BoolValue message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IBoolValue, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a BoolValue message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns BoolValue
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.BoolValue;

            /**
             * Decodes a BoolValue message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns BoolValue
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.BoolValue;

            /**
             * Verifies a BoolValue message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a BoolValue message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns BoolValue
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.BoolValue;

            /**
             * Creates a plain object from a BoolValue message. Also converts values to other types if specified.
             * @param message BoolValue
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.BoolValue, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this BoolValue to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for BoolValue
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        /** Properties of a StringValue. */
        interface IStringValue {

            /** StringValue value */
            value?: (string|null);
        }

        /** Represents a StringValue. */
        class StringValue implements IStringValue {

            /**
             * Constructs a new StringValue.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IStringValue);

            /** StringValue value. */
            public value: string;

            /**
             * Creates a new StringValue instance using the specified properties.
             * @param [properties] Properties to set
             * @returns StringValue instance
             */
            public static create(properties?: google.protobuf.IStringValue): google.protobuf.StringValue;

            /**
             * Encodes the specified StringValue message. Does not implicitly {@link google.protobuf.StringValue.verify|verify} messages.
             * @param message StringValue message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IStringValue, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified StringValue message, length delimited. Does not implicitly {@link google.protobuf.StringValue.verify|verify} messages.
             * @param message StringValue message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IStringValue, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a StringValue message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns StringValue
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.StringValue;

            /**
             * Decodes a StringValue message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns StringValue
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.StringValue;

            /**
             * Verifies a StringValue message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a StringValue message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns StringValue
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.StringValue;

            /**
             * Creates a plain object from a StringValue message. Also converts values to other types if specified.
             * @param message StringValue
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.StringValue, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this StringValue to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for StringValue
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        /** Properties of a BytesValue. */
        interface IBytesValue {

            /** BytesValue value */
            value?: (Uint8Array|null);
        }

        /** Represents a BytesValue. */
        class BytesValue implements IBytesValue {

            /**
             * Constructs a new BytesValue.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IBytesValue);

            /** BytesValue value. */
            public value: Uint8Array;

            /**
             * Creates a new BytesValue instance using the specified properties.
             * @param [properties] Properties to set
             * @returns BytesValue instance
             */
            public static create(properties?: google.protobuf.IBytesValue): google.protobuf.BytesValue;

            /**
             * Encodes the specified BytesValue message. Does not implicitly {@link google.protobuf.BytesValue.verify|verify} messages.
             * @param message BytesValue message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IBytesValue, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified BytesValue message, length delimited. Does not implicitly {@link google.protobuf.BytesValue.verify|verify} messages.
             * @param message BytesValue message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IBytesValue, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a BytesValue message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns BytesValue
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.BytesValue;

            /**
             * Decodes a BytesValue message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns BytesValue
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.BytesValue;

            /**
             * Verifies a BytesValue message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a BytesValue message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns BytesValue
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.BytesValue;

            /**
             * Creates a plain object from a BytesValue message. Also converts values to other types if specified.
             * @param message BytesValue
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.BytesValue, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this BytesValue to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for BytesValue
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }
    }
}
