#!/usr/bin/env python3
"""Демонстрационный читатель числовых формул из БД.
Не ядро Ebala. Не предназначен для формул-операторов, интегралов и нормативных выборок.
Никакого eval/exec над JSON. Только белый список AST.
"""
from pathlib import Path
import ast,math,json,operator

ROOT=Path(__file__).resolve().parents[1]
FORMULAS={f['id']:f for f in json.loads((ROOT/'data/formulas.json').read_text(encoding='utf-8'))['formulas']}

BIN={ast.Add:operator.add,ast.Sub:operator.sub,ast.Mult:operator.mul,ast.Div:operator.truediv,ast.Pow:operator.pow}
UNARY={ast.UAdd:operator.pos,ast.USub:operator.neg}
CALL={'sqrt':math.sqrt,'sin':math.sin,'cos':math.cos,'log10':math.log10,'abs':abs,'min':min,'max':max}
CONSTANTS={'pi':math.pi}
CMP={ast.LtE:operator.le,ast.Lt:operator.lt,ast.GtE:operator.ge,ast.Gt:operator.gt,ast.Eq:operator.eq}

def _calc(node,env):
    if isinstance(node,ast.Expression):return _calc(node.body,env)
    if isinstance(node,ast.Constant) and type(node.value) in (float,int):return float(node.value)
    if isinstance(node,ast.Name):
        if node.id in env and type(env[node.id]) in (float,int):return float(env[node.id])
        if node.id in CONSTANTS:return CONSTANTS[node.id]
        raise ValueError(f'Отсутствует числовой аргумент {node.id}')
    if isinstance(node,ast.BinOp) and type(node.op) in BIN:return BIN[type(node.op)](_calc(node.left,env),_calc(node.right,env))
    if isinstance(node,ast.UnaryOp) and type(node.op) in UNARY:return UNARY[type(node.op)](_calc(node.operand,env))
    if isinstance(node,ast.Compare) and len(node.ops)==1 and type(node.ops[0]) in CMP and len(node.comparators)==1:
        return CMP[type(node.ops[0])](_calc(node.left,env),_calc(node.comparators[0],env))
    if isinstance(node,ast.Call) and isinstance(node.func,ast.Name) and node.func.id in CALL and not node.keywords:
        return CALL[node.func.id](*[_calc(x,env) for x in node.args])
    raise ValueError('Неподдерживаемая математическая конструкция, нужен отдельный узел алгоритма')

def formula_value(formula_id,inputs):
    f=FORMULAS[formula_id]
    if f['kind'] not in ('formula','derived') or f['normative_status']=='FORMAL_REQUIRES_FE_AND_NUMERICAL_INTEGRATION':
        raise ValueError(f'Формула {formula_id} содержит выбор/интегрирование и не является скалярным выражением')
    val=_calc(ast.parse(f['expression'],mode='eval'),inputs)
    if not math.isfinite(val):raise ArithmeticError('Невозможно получить конечный результат')
    return val

def predicate_value(formula_id,inputs):
    f=FORMULAS[formula_id]
    if f['kind']!='predicate':raise ValueError('Ожидался логический критерий')
    if not set(f['inputs'])<=set(inputs):raise ValueError('Не хватает исходных данных для логического критерия')
    result=_calc(ast.parse(f['expression'],mode='eval'),inputs)
    if type(result) is not bool:raise ValueError('Критерий не вернул логическое значение')
    return result

if __name__=='__main__':
    x={'w0':380.0,'k':.85,'c_aerodynamic':.8}
    print('11.2',formula_value('11.2',x),'Па')
