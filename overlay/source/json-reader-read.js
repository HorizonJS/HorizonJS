    read : function(response){
        var o = Ext.decode(response.responseText);
        if(!o || typeof o != 'object'){
            throw new SyntaxError('JsonReader response must contain a data object or array');
        }
        if(o.metaData){
            var meta = o.metaData;
            if(typeof meta != 'object' || Ext.isArray(meta) || !Ext.isArray(meta.fields)){
                throw new SyntaxError('Invalid JsonReader metadata');
            }
            var properties = ['root', 'id', 'totalProperty', 'successProperty'];
            for(var i = 0; i < properties.length; i++){
                if(meta[properties[i]]){
                    this.getJsonAccessor(meta[properties[i]]);
                }
            }
            // Validate paths and field configuration before changing reader/store state.
            for(var i = 0; i < meta.fields.length; i++){
                var field = meta.fields[i];
                var name = typeof field == 'string' ? field : field && field.name;
                if(typeof name != 'string' || !name.length || name == '__proto__' || name == 'constructor' || name == 'prototype'){
                    throw new SyntaxError('Invalid JSON field name');
                }
                if(typeof field == 'object'){
                    if(!field || Ext.isArray(field) || Object.prototype.hasOwnProperty.call(field, '__proto__') || Object.prototype.hasOwnProperty.call(field, 'constructor') || Object.prototype.hasOwnProperty.call(field, 'prototype')){
                        throw new SyntaxError('Unsafe JSON field configuration');
                    }
                    if(field.convert !== undefined && typeof field.convert != 'function'){
                        throw new SyntaxError('JSON cannot supply field converter functions');
                    }
                }
                this.getJsonAccessor(field.mapping !== undefined && field.mapping !== null ? field.mapping : name);
            }
            var recordType = Ext.data.Record.create(meta.fields);
            delete this.ef;
            this.meta = meta;
            this.recordType = recordType;
            this.onMetaChange(meta, recordType, o);
        }
        return this.readRecords(o);
    },
