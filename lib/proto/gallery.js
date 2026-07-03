/*eslint-disable block-scoped-var, id-length, no-control-regex, no-magic-numbers, no-mixed-operators, no-prototype-builtins, no-redeclare, no-shadow, no-var, sort-vars, default-case, jsdoc/require-param*/
"use strict";

var $protobuf = require("protobufjs/minimal");

// Common aliases
var $Reader = $protobuf.Reader, $Writer = $protobuf.Writer, $util = $protobuf.util;
var $Object = $util.global.Object, $undefined = $util.global.undefined, $Error = $util.global.Error, $TypeError = $util.global.TypeError, $String = $util.global.String, $Number = $util.global.Number, $parseInt = $util.global.parseInt, $BigInt = $util.global.BigInt;

// Exported root namespace
var $root = $protobuf.roots["default"] || ($protobuf.roots["default"] = {});

$root.gallery = (function() {

    /**
     * Namespace gallery.
     * @exports gallery
     * @namespace
     */
    var gallery = {};

    gallery.Artwork = (function() {

        /**
         * Properties of an Artwork.
         * @typedef {Object} gallery.Artwork.$Properties
         * @property {string|null} [id] Artwork id
         * @property {string|null} [title] Artwork title
         * @property {string|null} [description] Artwork description
         * @property {string|null} [themeId] Artwork themeId
         * @property {number|Long|null} [timestamp] Artwork timestamp
         * @property {string|null} [imagePath] Artwork imagePath
         * @property {Array.<Uint8Array>} [$unknowns] Unknown fields preserved while decoding when enabled
         */

        /**
         * Properties of an Artwork.
         * @memberof gallery
         * @interface IArtwork
         * @augments gallery.Artwork.$Properties
         * @deprecated Use gallery.Artwork.$Properties instead.
         */

        /**
         * Shape of an Artwork.
         * @typedef {gallery.Artwork.$Properties} gallery.Artwork.$Shape
         */

        /**
         * Constructs a new Artwork.
         * @memberof gallery
         * @classdesc Represents an Artwork.
         * @constructor
         * @param {gallery.Artwork.$Properties=} [properties] Properties to set
         * @property {Array.<Uint8Array>} [$unknowns] Unknown fields preserved while decoding when enabled
         */
        var Artwork = function (properties) {
            if (properties)
                for (var keys = $Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null && keys[i] !== "__proto__")
                        this[keys[i]] = properties[keys[i]];
        };

        /**
         * Artwork id.
         * @member {string} id
         * @memberof gallery.Artwork
         * @instance
         */
        Artwork.prototype.id = "";

        /**
         * Artwork title.
         * @member {string} title
         * @memberof gallery.Artwork
         * @instance
         */
        Artwork.prototype.title = "";

        /**
         * Artwork description.
         * @member {string} description
         * @memberof gallery.Artwork
         * @instance
         */
        Artwork.prototype.description = "";

        /**
         * Artwork themeId.
         * @member {string} themeId
         * @memberof gallery.Artwork
         * @instance
         */
        Artwork.prototype.themeId = "";

        /**
         * Artwork timestamp.
         * @member {number|Long} timestamp
         * @memberof gallery.Artwork
         * @instance
         */
        Artwork.prototype.timestamp = $util.Long ? $util.Long.fromBits(0,0,false) : 0;

        /**
         * Artwork imagePath.
         * @member {string} imagePath
         * @memberof gallery.Artwork
         * @instance
         */
        Artwork.prototype.imagePath = "";

        /**
         * Creates a new Artwork instance using the specified properties.
         * @function create
         * @memberof gallery.Artwork
         * @static
         * @param {gallery.Artwork.$Properties=} [properties] Properties to set
         * @returns {gallery.Artwork} Artwork instance
         * @type {{
         *   (properties: gallery.Artwork.$Shape): gallery.Artwork & gallery.Artwork.$Shape;
         *   (properties?: gallery.Artwork.$Properties): gallery.Artwork;
         * }}
         */
        Artwork.create = function(properties) {
            return new Artwork(properties);
        };

        /**
         * Encodes the specified Artwork message. Does not implicitly {@link gallery.Artwork.verify|verify} messages.
         * @function encode
         * @memberof gallery.Artwork
         * @static
         * @param {gallery.Artwork.$Properties} message Artwork message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        Artwork.encode = function (message, writer, _depth) {
            if (!writer)
                writer = $Writer.create();
            if (_depth === $undefined)
                _depth = 0;
            if (_depth > $util.recursionLimit)
                throw $Error("max depth exceeded");
            if (message.id != null && $Object.hasOwnProperty.call(message, "id") && message.id !== "")
                writer.uint32(/* id 1, wireType 2 =*/10).string(message.id);
            if (message.title != null && $Object.hasOwnProperty.call(message, "title") && message.title !== "")
                writer.uint32(/* id 2, wireType 2 =*/18).string(message.title);
            if (message.description != null && $Object.hasOwnProperty.call(message, "description") && message.description !== "")
                writer.uint32(/* id 3, wireType 2 =*/26).string(message.description);
            if (message.themeId != null && $Object.hasOwnProperty.call(message, "themeId") && message.themeId !== "")
                writer.uint32(/* id 4, wireType 2 =*/34).string(message.themeId);
            if (message.timestamp != null && $Object.hasOwnProperty.call(message, "timestamp") && (typeof message.timestamp === "object" ? message.timestamp.low || message.timestamp.high : message.timestamp !== 0))
                writer.uint32(/* id 5, wireType 0 =*/40).int64(message.timestamp);
            if (message.imagePath != null && $Object.hasOwnProperty.call(message, "imagePath") && message.imagePath !== "")
                writer.uint32(/* id 6, wireType 2 =*/50).string(message.imagePath);
            if (message.$unknowns != null && $Object.hasOwnProperty.call(message, "$unknowns"))
                for (var i = 0; i < message.$unknowns.length; ++i)
                    writer.raw(message.$unknowns[i]);
            return writer;
        };

        /**
         * Encodes the specified Artwork message, length delimited. Does not implicitly {@link gallery.Artwork.verify|verify} messages.
         * @function encodeDelimited
         * @memberof gallery.Artwork
         * @static
         * @param {gallery.Artwork.$Properties} message Artwork message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        Artwork.encodeDelimited = function(message, writer) {
            return this.encode(message, writer && writer.len ? writer.fork() : writer).ldelim();
        };

        /**
         * Decodes an Artwork message from the specified reader or buffer.
         * @function decode
         * @memberof gallery.Artwork
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {gallery.Artwork & gallery.Artwork.$Shape} Artwork
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        Artwork.decode = function (reader, length, _end, _depth, _target) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            if (_depth === $undefined)
                _depth = 0;
            if (_depth > $Reader.recursionLimit)
                throw $Error("max depth exceeded");
            var end = length === $undefined ? reader.len : reader.pos + length, message = _target || new $root.gallery.Artwork(), value;
            while (reader.pos < end) {
                var start = reader.pos;
                var tag = reader.tag();
                if (tag === _end) {
                    _end = $undefined;
                    break;
                }
                var wireType = tag & 7;
                switch (tag >>>= 3) {
                case 1: {
                        if (wireType !== 2)
                            break;
                        if ((value = reader.stringVerify()).length)
                            message.id = value;
                        else
                            delete message.id;
                        continue;
                    }
                case 2: {
                        if (wireType !== 2)
                            break;
                        if ((value = reader.stringVerify()).length)
                            message.title = value;
                        else
                            delete message.title;
                        continue;
                    }
                case 3: {
                        if (wireType !== 2)
                            break;
                        if ((value = reader.stringVerify()).length)
                            message.description = value;
                        else
                            delete message.description;
                        continue;
                    }
                case 4: {
                        if (wireType !== 2)
                            break;
                        if ((value = reader.stringVerify()).length)
                            message.themeId = value;
                        else
                            delete message.themeId;
                        continue;
                    }
                case 5: {
                        if (wireType !== 0)
                            break;
                        if (typeof (value = reader.int64()) === "object" ? value.low || value.high : value !== 0)
                            message.timestamp = value;
                        else
                            delete message.timestamp;
                        continue;
                    }
                case 6: {
                        if (wireType !== 2)
                            break;
                        if ((value = reader.stringVerify()).length)
                            message.imagePath = value;
                        else
                            delete message.imagePath;
                        continue;
                    }
                }
                reader.skipType(wireType, _depth, tag);
                if (!reader.discardUnknown) {
                    $util.makeProp(message, "$unknowns", false);
                    (message.$unknowns || (message.$unknowns = [])).push(reader.raw(start, reader.pos));
                }
            }
            if (_end !== $undefined)
                throw $Error("missing end group");
            return message;
        };

        /**
         * Decodes an Artwork message from the specified reader or buffer, length delimited.
         * @function decodeDelimited
         * @memberof gallery.Artwork
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @returns {gallery.Artwork & gallery.Artwork.$Shape} Artwork
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        Artwork.decodeDelimited = function(reader) {
            if (!(reader instanceof $Reader))
                reader = new $Reader(reader);
            return this.decode(reader, reader.uint32());
        };

        /**
         * Verifies an Artwork message.
         * @function verify
         * @memberof gallery.Artwork
         * @static
         * @param {Object.<string,*>} message Plain object to verify
         * @returns {string|null} `null` if valid, otherwise the reason why it is not
         */
        Artwork.verify = function (message, _depth) {
            if (typeof message !== "object" || message === null)
                return "object expected";
            if (_depth === $undefined)
                _depth = 0;
            if (_depth > $util.recursionLimit)
                return "max depth exceeded";
            if (message.id != null && $Object.hasOwnProperty.call(message, "id"))
                if (!$util.isString(message.id))
                    return "id: string expected";
            if (message.title != null && $Object.hasOwnProperty.call(message, "title"))
                if (!$util.isString(message.title))
                    return "title: string expected";
            if (message.description != null && $Object.hasOwnProperty.call(message, "description"))
                if (!$util.isString(message.description))
                    return "description: string expected";
            if (message.themeId != null && $Object.hasOwnProperty.call(message, "themeId"))
                if (!$util.isString(message.themeId))
                    return "themeId: string expected";
            if (message.timestamp != null && $Object.hasOwnProperty.call(message, "timestamp"))
                if (!$util.isInteger(message.timestamp) && !(message.timestamp && $util.isInteger(message.timestamp.low) && $util.isInteger(message.timestamp.high)))
                    return "timestamp: integer|Long expected";
            if (message.imagePath != null && $Object.hasOwnProperty.call(message, "imagePath"))
                if (!$util.isString(message.imagePath))
                    return "imagePath: string expected";
            return null;
        };

        /**
         * Creates an Artwork message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof gallery.Artwork
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {gallery.Artwork} Artwork
         */
        Artwork.fromObject = function (object, _depth) {
            if (object instanceof $root.gallery.Artwork)
                return object;
            if (!$util.isObject(object))
                throw $TypeError(".gallery.Artwork: object expected");
            if (_depth === $undefined)
                _depth = 0;
            if (_depth > $util.recursionLimit)
                throw $Error("max depth exceeded");
            var message = new $root.gallery.Artwork();
            if (object.id != null)
                if (typeof object.id !== "string" || object.id.length)
                    message.id = $String(object.id);
            if (object.title != null)
                if (typeof object.title !== "string" || object.title.length)
                    message.title = $String(object.title);
            if (object.description != null)
                if (typeof object.description !== "string" || object.description.length)
                    message.description = $String(object.description);
            if (object.themeId != null)
                if (typeof object.themeId !== "string" || object.themeId.length)
                    message.themeId = $String(object.themeId);
            if (object.timestamp != null)
                if (typeof object.timestamp === "object" ? object.timestamp.low || object.timestamp.high : $Number(object.timestamp) !== 0)
                    if ($util.Long)
                        message.timestamp = $util.Long.fromValue(object.timestamp, false);
                    else if (typeof object.timestamp === "string")
                        message.timestamp = $parseInt(object.timestamp, 10);
                    else if (typeof object.timestamp === "number")
                        message.timestamp = object.timestamp;
                    else if (typeof object.timestamp === "object")
                        message.timestamp = new $util.LongBits(object.timestamp.low >>> 0, object.timestamp.high >>> 0).toNumber();
            if (object.imagePath != null)
                if (typeof object.imagePath !== "string" || object.imagePath.length)
                    message.imagePath = $String(object.imagePath);
            return message;
        };

        /**
         * Creates a plain object from an Artwork message. Also converts values to other types if specified.
         * @function toObject
         * @memberof gallery.Artwork
         * @static
         * @param {gallery.Artwork} message Artwork
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        Artwork.toObject = function (message, options, _depth) {
            if (!options)
                options = {};
            if (_depth === $undefined)
                _depth = 0;
            if (_depth > $util.recursionLimit)
                throw $Error("max depth exceeded");
            var object = {};
            if (options.defaults) {
                object.id = "";
                object.title = "";
                object.description = "";
                object.themeId = "";
                if ($util.Long) {
                    var long = new $util.Long(0, 0, false);
                    object.timestamp = options.longs === $String ? long.toString() : options.longs === $Number ? long.toNumber() : typeof $BigInt !== "undefined" && options.longs === $BigInt ? long.toBigInt() : long;
                } else
                    object.timestamp = options.longs === $String ? "0" : typeof $BigInt !== "undefined" && options.longs === $BigInt ? $BigInt("0") : 0;
                object.imagePath = "";
            }
            if (message.id != null && $Object.hasOwnProperty.call(message, "id"))
                object.id = message.id;
            if (message.title != null && $Object.hasOwnProperty.call(message, "title"))
                object.title = message.title;
            if (message.description != null && $Object.hasOwnProperty.call(message, "description"))
                object.description = message.description;
            if (message.themeId != null && $Object.hasOwnProperty.call(message, "themeId"))
                object.themeId = message.themeId;
            if (message.timestamp != null && $Object.hasOwnProperty.call(message, "timestamp"))
                if (typeof $BigInt !== "undefined" && options.longs === $BigInt)
                    object.timestamp = typeof message.timestamp === "number" ? $BigInt(message.timestamp) : $util.Long.fromBits(message.timestamp.low >>> 0, message.timestamp.high >>> 0, false).toBigInt();
                else if (typeof message.timestamp === "number")
                    object.timestamp = options.longs === $String ? $String(message.timestamp) : message.timestamp;
                else
                    object.timestamp = options.longs === $String ? $util.Long.prototype.toString.call(message.timestamp) : options.longs === $Number ? new $util.LongBits(message.timestamp.low >>> 0, message.timestamp.high >>> 0).toNumber() : message.timestamp;
            if (message.imagePath != null && $Object.hasOwnProperty.call(message, "imagePath"))
                object.imagePath = message.imagePath;
            return object;
        };

        /**
         * Converts this Artwork to JSON.
         * @function toJSON
         * @memberof gallery.Artwork
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        Artwork.prototype.toJSON = function() {
            return Artwork.toObject(this, $protobuf.util.toJSONOptions);
        };

        /**
         * Gets the type url for Artwork
         * @function getTypeUrl
         * @memberof gallery.Artwork
         * @static
         * @param {string} [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns {string} The type url
         */
        Artwork.getTypeUrl = function(prefix) {
            if (prefix === $undefined)
                prefix = "type.googleapis.com";
            return prefix + "/gallery.Artwork";
        };

        return Artwork;
    })();

    gallery.Theme = (function() {

        /**
         * Properties of a Theme.
         * @typedef {Object} gallery.Theme.$Properties
         * @property {string|null} [id] Theme id
         * @property {string|null} [name] Theme name
         * @property {string|null} [description] Theme description
         * @property {Array.<Uint8Array>} [$unknowns] Unknown fields preserved while decoding when enabled
         */

        /**
         * Properties of a Theme.
         * @memberof gallery
         * @interface ITheme
         * @augments gallery.Theme.$Properties
         * @deprecated Use gallery.Theme.$Properties instead.
         */

        /**
         * Shape of a Theme.
         * @typedef {gallery.Theme.$Properties} gallery.Theme.$Shape
         */

        /**
         * Constructs a new Theme.
         * @memberof gallery
         * @classdesc Represents a Theme.
         * @constructor
         * @param {gallery.Theme.$Properties=} [properties] Properties to set
         * @property {Array.<Uint8Array>} [$unknowns] Unknown fields preserved while decoding when enabled
         */
        var Theme = function (properties) {
            if (properties)
                for (var keys = $Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null && keys[i] !== "__proto__")
                        this[keys[i]] = properties[keys[i]];
        };

        /**
         * Theme id.
         * @member {string} id
         * @memberof gallery.Theme
         * @instance
         */
        Theme.prototype.id = "";

        /**
         * Theme name.
         * @member {string} name
         * @memberof gallery.Theme
         * @instance
         */
        Theme.prototype.name = "";

        /**
         * Theme description.
         * @member {string} description
         * @memberof gallery.Theme
         * @instance
         */
        Theme.prototype.description = "";

        /**
         * Creates a new Theme instance using the specified properties.
         * @function create
         * @memberof gallery.Theme
         * @static
         * @param {gallery.Theme.$Properties=} [properties] Properties to set
         * @returns {gallery.Theme} Theme instance
         * @type {{
         *   (properties: gallery.Theme.$Shape): gallery.Theme & gallery.Theme.$Shape;
         *   (properties?: gallery.Theme.$Properties): gallery.Theme;
         * }}
         */
        Theme.create = function(properties) {
            return new Theme(properties);
        };

        /**
         * Encodes the specified Theme message. Does not implicitly {@link gallery.Theme.verify|verify} messages.
         * @function encode
         * @memberof gallery.Theme
         * @static
         * @param {gallery.Theme.$Properties} message Theme message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        Theme.encode = function (message, writer, _depth) {
            if (!writer)
                writer = $Writer.create();
            if (_depth === $undefined)
                _depth = 0;
            if (_depth > $util.recursionLimit)
                throw $Error("max depth exceeded");
            if (message.id != null && $Object.hasOwnProperty.call(message, "id") && message.id !== "")
                writer.uint32(/* id 1, wireType 2 =*/10).string(message.id);
            if (message.name != null && $Object.hasOwnProperty.call(message, "name") && message.name !== "")
                writer.uint32(/* id 2, wireType 2 =*/18).string(message.name);
            if (message.description != null && $Object.hasOwnProperty.call(message, "description") && message.description !== "")
                writer.uint32(/* id 3, wireType 2 =*/26).string(message.description);
            if (message.$unknowns != null && $Object.hasOwnProperty.call(message, "$unknowns"))
                for (var i = 0; i < message.$unknowns.length; ++i)
                    writer.raw(message.$unknowns[i]);
            return writer;
        };

        /**
         * Encodes the specified Theme message, length delimited. Does not implicitly {@link gallery.Theme.verify|verify} messages.
         * @function encodeDelimited
         * @memberof gallery.Theme
         * @static
         * @param {gallery.Theme.$Properties} message Theme message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        Theme.encodeDelimited = function(message, writer) {
            return this.encode(message, writer && writer.len ? writer.fork() : writer).ldelim();
        };

        /**
         * Decodes a Theme message from the specified reader or buffer.
         * @function decode
         * @memberof gallery.Theme
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {gallery.Theme & gallery.Theme.$Shape} Theme
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        Theme.decode = function (reader, length, _end, _depth, _target) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            if (_depth === $undefined)
                _depth = 0;
            if (_depth > $Reader.recursionLimit)
                throw $Error("max depth exceeded");
            var end = length === $undefined ? reader.len : reader.pos + length, message = _target || new $root.gallery.Theme(), value;
            while (reader.pos < end) {
                var start = reader.pos;
                var tag = reader.tag();
                if (tag === _end) {
                    _end = $undefined;
                    break;
                }
                var wireType = tag & 7;
                switch (tag >>>= 3) {
                case 1: {
                        if (wireType !== 2)
                            break;
                        if ((value = reader.stringVerify()).length)
                            message.id = value;
                        else
                            delete message.id;
                        continue;
                    }
                case 2: {
                        if (wireType !== 2)
                            break;
                        if ((value = reader.stringVerify()).length)
                            message.name = value;
                        else
                            delete message.name;
                        continue;
                    }
                case 3: {
                        if (wireType !== 2)
                            break;
                        if ((value = reader.stringVerify()).length)
                            message.description = value;
                        else
                            delete message.description;
                        continue;
                    }
                }
                reader.skipType(wireType, _depth, tag);
                if (!reader.discardUnknown) {
                    $util.makeProp(message, "$unknowns", false);
                    (message.$unknowns || (message.$unknowns = [])).push(reader.raw(start, reader.pos));
                }
            }
            if (_end !== $undefined)
                throw $Error("missing end group");
            return message;
        };

        /**
         * Decodes a Theme message from the specified reader or buffer, length delimited.
         * @function decodeDelimited
         * @memberof gallery.Theme
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @returns {gallery.Theme & gallery.Theme.$Shape} Theme
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        Theme.decodeDelimited = function(reader) {
            if (!(reader instanceof $Reader))
                reader = new $Reader(reader);
            return this.decode(reader, reader.uint32());
        };

        /**
         * Verifies a Theme message.
         * @function verify
         * @memberof gallery.Theme
         * @static
         * @param {Object.<string,*>} message Plain object to verify
         * @returns {string|null} `null` if valid, otherwise the reason why it is not
         */
        Theme.verify = function (message, _depth) {
            if (typeof message !== "object" || message === null)
                return "object expected";
            if (_depth === $undefined)
                _depth = 0;
            if (_depth > $util.recursionLimit)
                return "max depth exceeded";
            if (message.id != null && $Object.hasOwnProperty.call(message, "id"))
                if (!$util.isString(message.id))
                    return "id: string expected";
            if (message.name != null && $Object.hasOwnProperty.call(message, "name"))
                if (!$util.isString(message.name))
                    return "name: string expected";
            if (message.description != null && $Object.hasOwnProperty.call(message, "description"))
                if (!$util.isString(message.description))
                    return "description: string expected";
            return null;
        };

        /**
         * Creates a Theme message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof gallery.Theme
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {gallery.Theme} Theme
         */
        Theme.fromObject = function (object, _depth) {
            if (object instanceof $root.gallery.Theme)
                return object;
            if (!$util.isObject(object))
                throw $TypeError(".gallery.Theme: object expected");
            if (_depth === $undefined)
                _depth = 0;
            if (_depth > $util.recursionLimit)
                throw $Error("max depth exceeded");
            var message = new $root.gallery.Theme();
            if (object.id != null)
                if (typeof object.id !== "string" || object.id.length)
                    message.id = $String(object.id);
            if (object.name != null)
                if (typeof object.name !== "string" || object.name.length)
                    message.name = $String(object.name);
            if (object.description != null)
                if (typeof object.description !== "string" || object.description.length)
                    message.description = $String(object.description);
            return message;
        };

        /**
         * Creates a plain object from a Theme message. Also converts values to other types if specified.
         * @function toObject
         * @memberof gallery.Theme
         * @static
         * @param {gallery.Theme} message Theme
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        Theme.toObject = function (message, options, _depth) {
            if (!options)
                options = {};
            if (_depth === $undefined)
                _depth = 0;
            if (_depth > $util.recursionLimit)
                throw $Error("max depth exceeded");
            var object = {};
            if (options.defaults) {
                object.id = "";
                object.name = "";
                object.description = "";
            }
            if (message.id != null && $Object.hasOwnProperty.call(message, "id"))
                object.id = message.id;
            if (message.name != null && $Object.hasOwnProperty.call(message, "name"))
                object.name = message.name;
            if (message.description != null && $Object.hasOwnProperty.call(message, "description"))
                object.description = message.description;
            return object;
        };

        /**
         * Converts this Theme to JSON.
         * @function toJSON
         * @memberof gallery.Theme
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        Theme.prototype.toJSON = function() {
            return Theme.toObject(this, $protobuf.util.toJSONOptions);
        };

        /**
         * Gets the type url for Theme
         * @function getTypeUrl
         * @memberof gallery.Theme
         * @static
         * @param {string} [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns {string} The type url
         */
        Theme.getTypeUrl = function(prefix) {
            if (prefix === $undefined)
                prefix = "type.googleapis.com";
            return prefix + "/gallery.Theme";
        };

        return Theme;
    })();

    return gallery;
})();

module.exports = $root;
