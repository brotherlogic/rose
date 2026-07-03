import * as $protobuf from "protobufjs";
import Long = require("long");

/** Namespace gallery. */
export namespace gallery {

    /**
     * Properties of an Artwork.
     * @deprecated Use gallery.Artwork.$Properties instead.
     */
    interface IArtwork extends gallery.Artwork.$Properties {
    }

    /** Represents an Artwork. */
    class Artwork {

        /**
         * Constructs a new Artwork.
         * @param [properties] Properties to set
         */
        constructor(properties?: gallery.Artwork.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** Artwork id. */
        id: string;

        /** Artwork title. */
        title: string;

        /** Artwork description. */
        description: string;

        /** Artwork themeId. */
        themeId: string;

        /** Artwork timestamp. */
        timestamp: (number|Long);

        /** Artwork imagePath. */
        imagePath: string;

        /**
         * Creates a new Artwork instance using the specified properties.
         * @param [properties] Properties to set
         * @returns Artwork instance
         */
        static create(properties: gallery.Artwork.$Shape): gallery.Artwork & gallery.Artwork.$Shape;
        static create(properties?: gallery.Artwork.$Properties): gallery.Artwork;

        /**
         * Encodes the specified Artwork message. Does not implicitly {@link gallery.Artwork.verify|verify} messages.
         * @param message Artwork message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: gallery.Artwork.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified Artwork message, length delimited. Does not implicitly {@link gallery.Artwork.verify|verify} messages.
         * @param message Artwork message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: gallery.Artwork.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes an Artwork message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {gallery.Artwork & gallery.Artwork.$Shape} Artwork
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): gallery.Artwork & gallery.Artwork.$Shape;

        /**
         * Decodes an Artwork message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {gallery.Artwork & gallery.Artwork.$Shape} Artwork
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): gallery.Artwork & gallery.Artwork.$Shape;

        /**
         * Verifies an Artwork message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates an Artwork message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns Artwork
         */
        static fromObject(object: { [k: string]: any }): gallery.Artwork;

        /**
         * Creates a plain object from an Artwork message. Also converts values to other types if specified.
         * @param message Artwork
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: gallery.Artwork, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this Artwork to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for Artwork
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace Artwork {

        /** Properties of an Artwork. */
        interface $Properties {

            /** Artwork id */
            id?: (string|null);

            /** Artwork title */
            title?: (string|null);

            /** Artwork description */
            description?: (string|null);

            /** Artwork themeId */
            themeId?: (string|null);

            /** Artwork timestamp */
            timestamp?: (number|Long|null);

            /** Artwork imagePath */
            imagePath?: (string|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of an Artwork. */
        type $Shape = gallery.Artwork.$Properties;
    }

    /**
     * Properties of a Theme.
     * @deprecated Use gallery.Theme.$Properties instead.
     */
    interface ITheme extends gallery.Theme.$Properties {
    }

    /** Represents a Theme. */
    class Theme {

        /**
         * Constructs a new Theme.
         * @param [properties] Properties to set
         */
        constructor(properties?: gallery.Theme.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** Theme id. */
        id: string;

        /** Theme name. */
        name: string;

        /** Theme description. */
        description: string;

        /**
         * Creates a new Theme instance using the specified properties.
         * @param [properties] Properties to set
         * @returns Theme instance
         */
        static create(properties: gallery.Theme.$Shape): gallery.Theme & gallery.Theme.$Shape;
        static create(properties?: gallery.Theme.$Properties): gallery.Theme;

        /**
         * Encodes the specified Theme message. Does not implicitly {@link gallery.Theme.verify|verify} messages.
         * @param message Theme message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: gallery.Theme.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified Theme message, length delimited. Does not implicitly {@link gallery.Theme.verify|verify} messages.
         * @param message Theme message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: gallery.Theme.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a Theme message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {gallery.Theme & gallery.Theme.$Shape} Theme
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): gallery.Theme & gallery.Theme.$Shape;

        /**
         * Decodes a Theme message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {gallery.Theme & gallery.Theme.$Shape} Theme
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): gallery.Theme & gallery.Theme.$Shape;

        /**
         * Verifies a Theme message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a Theme message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns Theme
         */
        static fromObject(object: { [k: string]: any }): gallery.Theme;

        /**
         * Creates a plain object from a Theme message. Also converts values to other types if specified.
         * @param message Theme
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: gallery.Theme, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this Theme to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for Theme
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace Theme {

        /** Properties of a Theme. */
        interface $Properties {

            /** Theme id */
            id?: (string|null);

            /** Theme name */
            name?: (string|null);

            /** Theme description */
            description?: (string|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a Theme. */
        type $Shape = gallery.Theme.$Properties;
    }
}
